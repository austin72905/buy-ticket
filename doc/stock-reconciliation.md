# 庫存 Reconciliation 設計

這份文件說明 Redis stock 與 PostgreSQL section 庫存之間的 reconciliation。

## 1. 為什麼需要

目前 reservation 流程會同時碰到兩個系統：

1. Redis stock：用 Lua script 原子扣減，擋高併發超賣。
2. PostgreSQL：保存 section、reservation、order 的正式狀態。

正常流程：

```text
Redis reserve 成功
DB section.reserved_quantity 增加
DB 建立 reservation
DB commit
```

如果 Redis reserve 成功，但 DB transaction 失敗，程式會嘗試補償：

```text
Redis release 補回剛剛扣掉的數量
```

但是補償本身也可能失敗，例如：

- Redis timeout
- network error
- service crash

這時可能出現：

```text
Redis stock 少 1
DB 沒有對應 reservation
```

結果是票被卡住，使用者買不到，但 DB 看起來其實還有票。

Reconciliation 的目的就是定期以 DB 為準，修正 Redis stock 的偏差。

## 2. 目前實作

目前已新增 `stock-reconcile` background job。

庫存寫入目前有兩層保護：

1. Redis Lua script 是高併發請求的快速 admission gate，原子檢查並扣減可用庫存。
2. PostgreSQL 使用條件式原子 `UPDATE` 處理 reserve、release 與 confirm sale，避免 DB 庫存超賣、負數或確認超過已保留數量。

Reserve 的 DB 更新會確認：

```text
quantity > 0
quantity <= purchase_limit（purchase_limit 不為 0 時）
reserved_quantity + sold_quantity + quantity <= total_quantity
```

Release 與 confirm sale 則會確認 `reserved_quantity >= quantity`。因此 PostgreSQL 是 Redis 之外的第二道庫存防線，也是最終的 durable consistency boundary。

排程：

```text
0 * * * * *
```

意思是每分鐘第 0 秒執行一次。

流程：

1. 從 DB 讀出所有 event sections。
2. 用 DB 欄位計算可用數量：

```text
available = total_quantity - reserved_quantity - sold_quantity
```

3. 讀 Redis stock key。
4. 如果 Redis 數量和 DB available 不一致，就把 Redis 修正成 DB available。
5. 記錄檢查了幾個 section、修正了幾個 section。

相關程式：

- `BookingService.ReconcileStock`
- `RedisStockStore.ReconcileAll`
- background job：`stock-reconcile`

## 3. 和 RebuildStock 的差異

`RebuildStock`：

- `APP_ROLE=all` 或 `APP_ROLE=scheduler` 啟動時執行；純 `APP_ROLE=api` 不會執行。
- 直接用 DB 狀態重建所有 Redis stock。
- 適合服務啟動、Redis 清空、初始化。

`ReconcileStock`：

- background job 定期執行。
- 用 DB 狀態檢查並修正 Redis 偏差。
- 適合修復補償失敗造成的 Redis / DB 不一致。

## 4. 注意事項

Redis 和 DB 不是同一個 transaction，所以這不是絕對強一致。

Reconciliation 是「最終一致」修復手段。

要注意：

- 如果 reconciliation 剛好在 Redis reserve 成功、DB commit 還沒完成時執行，可能短暫把 Redis 修回 DB 舊值。
- 目前每分鐘跑一次，降低撞到 in-flight reservation 的機率。
- 目前 scheduler guard 只限制單一 process 內不重入；多個 scheduler Pod 之間沒有 distributed lock。
- Production 若要執行多個 scheduler replicas，可以加入 distributed lock、leader election 或更細緻的 reconciliation claim 機制。

目前作品階段的定位：

- Redis Lua 負責高併發扣庫存。
- PostgreSQL 條件式原子更新與 DB transaction 負責正式資料狀態及第二道庫存檢查。
- 失敗補償負責即時修復。
- `stock-reconcile` 負責補償失敗後的定期修復。

## 5. 後續可補強

- Redis release script 加上最大可用量上限，避免 release 過量。
- Reconciliation 加分散式鎖，避免多 instance 同時執行。
- Reconciliation 加 metrics，例如 checked/fixed/error count。
- 補一個 admin endpoint 或 CLI 手動觸發 reconciliation。
