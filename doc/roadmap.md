# buy-ticket Roadmap

## 1. 目標

這份文件用來對齊兩件事：

- `D:\ObsidianVault\KnowledgeBase\Excalidraw\buy-ticket 架構圖.md` 的完整搶票架構
- 目前 `D:\SourceCode\Go\buy-ticket` 已經落地的程式範圍

目前專案只完成了「搶到名額後的交易流程骨架」，還沒完成「高併發排隊與 Redis 控流」那一層。

---

## 2. 目前已完成

### 2.1 Domain

已完成的核心模型：

- `Event`
- `Section`
- `Reservation`
- `Order`
- `Payment`

已完成的核心規則：

- 活動是否在開賣期間
- 票區是否可保留
- 票區保留 / 釋放 / 確認售出
- reservation 確認 / 過期 / 取消
- order 付款 / 過期 / 取消
- payment 成功 / 失敗

### 2.2 Service

已完成的交易流程：

- `ReserveTicket`
- `CreateOrder`
- `PayOrder`
- `ExpireReservation`
- `CancelReservation`

已完成的交易處理方式：

- `BookingService.withTx(...)`
- memory 模式不開 transaction
- postgres 模式走 `pgx` transaction

### 2.3 Controller / API

目前已提供的 API：

- `POST /reservations`
- `POST /orders`
- `POST /payments`
- `POST /reservations/expire`
- `POST /reservations/cancel`
- `GET /healthz`

### 2.4 Persistence

已完成：

- PostgreSQL migration
- `sqlc.yaml`
- `db/query.sql`
- `sqlc generate`
- memory repository
- postgres repository

### 2.5 測試

已完成：

- `domain` 單元測試
- `service` 單元測試

---

## 3. 目前還沒完成

這些是架構圖裡有，但程式目前還沒有的部分。

### 3.1 Queue / Waiting Room

未完成：

- `precheck`
- `join queue`
- `queue status polling`
- `leave queue`
- `purchase token`

目前缺少的能力：

- Redis queue 結構
- queue token / purchase token 驗證
- 排隊放行 worker
- 防重複排隊

### 3.2 高併發控流

未完成：

- rate limiter
- token bucket
- Redis Lua 原子扣減
- 限購計數
- queue dedupe

### 3.3 非同步流程

未完成：

- RabbitMQ publisher / consumer
- 延遲過期處理
- DLX / expire queue
- callback 型付款後續處理

### 3.4 Read API

未完成或只存在於圖上：

- `GET /api/events/{eventId}`
- `GET /api/events/{eventId}/sections`
- `GET /api/events/{eventId}/availability`
- `GET /api/orders/{orderId}`
- queue 相關查詢 API

---

## 4. 當前專案定位

目前專案應該被視為：

**交易核心雛形**

更精確地說，是這條線已初步落地：

1. 保留票券
2. 建立訂單
3. 付款
4. 逾時釋放
5. 主動取消

還不能被視為完整的「高併發搶票系統」。

因為完整搶票系統至少還需要：

1. 排隊層
2. Redis 控流層
3. 非同步 worker 層
4. callback / 失敗補償

---

## 5. 建議開發順序

### Phase 1: 補齊查詢 API

目標：

- 讓基本活動 / 票區查詢可用

建議項目：

- `GET /events/:id`
- `GET /events/:id/sections`
- `GET /events/:id/availability`
- `GET /orders/:id`

### Phase 2: 補齊 Postgres 實際運行環境

目標：

- 讓 postgres 模式可直接本機啟動

建議項目：

- `.env.example`
- `docker-compose.yml`
- migration 指令說明
- sqlc generate 指令說明

### Phase 3: 導入 Redis 與 Queue

目標：

- 開始接近真正搶票架構

建議項目：

- queue token
- join queue
- queue status
- purchase token
- queue dedupe

### Phase 4: Redis Lua 原子保留

目標：

- 將高併發庫存保留從 DB 前移到 Redis

建議項目：

- `stock:event:{eventId}`
- `hold:order:{orderId}`
- `user:event:{eventId}:limit:{userId}`
- Lua script 做保留 / 釋放 / 限購判斷

### Phase 5: RabbitMQ 與過期清理

目標：

- 把逾時、補償、非同步流程補完整

建議項目：

- hold order expire event
- DLX / delay queue
- payment callback 後續處理
- 庫存補償

---

## 6. 實作原則

接下來維持這些原則：

- `domain` 保持乾淨，只放業務規則需要的欄位
- DB 快照與查詢最佳化資料留在 persistence / read model
- `repository` 對外回傳 `domain struct`
- `service` 負責流程與 transaction 邊界
- 小而清楚的 helper 優先，避免過度設計

---

## 7. 下一步建議

如果依照目前專案狀態，最建議的下一步是：

1. 補查詢 API
2. 補 postgres 啟動文件
3. 再開始做 queue / redis

原因是：

- 交易核心已經有骨架
- 先補查詢與啟動方式，系統比較容易跑起來
- 之後再進入 queue / redis，會比較穩
