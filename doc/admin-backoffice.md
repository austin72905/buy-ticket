# 後台管理功能規劃

這份文件整理 `buy-ticket` 後台的實作方向。目標不是一次做完整營運後台，而是分階段補上能展示工程能力、又能貼近目前購票系統資料模型的功能。

---

## 1. 目標

後台要解決三件事：

1. 讓管理者可以查看活動、票區、訂單與付款狀態。
2. 讓大量資料列表使用 keyset pagination，避免 `OFFSET` 深分頁效能退化。
3. 讓敏感資料預設脫敏，完整資料 reveal 需要權限與 audit log。

後台第一版不追求完整 CMS，而是補出作品亮點：

- 訂單列表 keyset pagination
- 票區庫存狀態查詢
- 後台角色權限
- 敏感資料脫敏
- reveal audit log

---

## 2. 目前系統可直接利用的資料

目前已存在的核心資料表：

- `users`
- `events`
- `event_sections`
- `reservations`
- `orders`
- `payments`
- `payment_attempts`
- `idempotency_keys`

這代表後台第一版可以先從「查詢與管控」開始，不需要先重做主流程。

---

## 3. 後台角色

建議先做兩種角色：

### 3.1 SUPER_ADMIN

- 可以查看所有後台頁面。
- 預設仍只看到脫敏資料。
- 可以呼叫 reveal API 查看完整敏感資料。
- 每次 reveal 都必須填寫原因並寫入 audit log。

### 3.2 EVENT_ADMIN

- 可以查看活動、票區、訂單。
- 永遠只能看到脫敏資料。
- 不能 reveal 完整手機、email、姓名或其他敏感資料。

---

## 4. 後台核心頁面

### 4.1 活動管理

第一版功能：

- 活動列表
- 活動詳情
- 活動狀態
- 開售時間
- 結束時間

後續功能：

- 建立活動
- 編輯活動
- 上架 / 下架
- 開售 / 結束

建議狀態：

- `draft`
- `published`
- `selling`
- `ended`

目前 `events` 已有活動資料，第一階段可以先做查詢與狀態調整。

---

### 4.2 票區 / 票種管理

目前系統的 `event_sections` 可以視為票區或票種。

第一版功能：

- 票區列表
- 票區名稱
- 價格
- 總庫存
- 鎖定中庫存
- 已售庫存
- 剩餘庫存

計算方式：

```sql
available_quantity = total_quantity - reserved_quantity - sold_quantity
```

後續功能：

- 修改總庫存
- 修改價格
- 設定每人限購

注意：如果要支援「每人限購」，需要新增欄位，例如：

```sql
ALTER TABLE event_sections
ADD COLUMN purchase_limit_per_user INTEGER NULL;
```

---

### 4.3 訂單管理

這是第一版最值得先做的後台功能。

列表欄位：

- order id
- order no
- user id
- event id
- section id
- quantity
- total amount
- status
- expires at
- created at
- updated at

訂單狀態：

- `pending_payment`
- `paid`
- `expired`
- `cancelled`

查詢條件：

- event id
- user id
- order status
- created time range

分頁方式使用 keyset pagination：

```sql
SELECT *
FROM orders
WHERE ($1::timestamptz IS NULL OR created_at < $1)
   OR (created_at = $1 AND id < $2)
ORDER BY created_at DESC, id DESC
LIMIT $3;
```

API response：

```json
{
  "items": [],
  "next_cursor": {
    "created_at": "2026-06-28T12:00:00Z",
    "id": 12345
  }
}
```

---

### 4.4 搶票請求紀錄

這是第二階段功能，不建議第一版就做。

原因：

- 搶票請求資料量會很大。
- 如果同步寫 DB，可能拖慢高併發主流程。
- 比較適合後續用 async log、queue 或批次寫入。

未來可以記錄：

- request id
- user id
- event id
- section id
- result
- latency
- error reason
- created at

結果類型：

- `success`
- `sold_out`
- `duplicate_purchase`
- `rate_limited`
- `queue_waiting`
- `purchase_token_invalid`
- `reservation_failed`

這張表適合 keyset pagination。

---

### 4.5 庫存流水

這是第二階段功能，適合在主流程穩定後補。

用途：

- 追蹤庫存為什麼被扣。
- 追蹤庫存為什麼被回補。
- Debug Redis / DB 庫存不一致。
- 展示補償與 reconciliation 能力。

未來可以新增 `stock_ledger_entries`：

```sql
CREATE TABLE stock_ledger_entries (
    id BIGSERIAL PRIMARY KEY,
    event_id BIGINT NOT NULL,
    section_id BIGINT NOT NULL,
    reservation_id BIGINT NULL,
    order_id BIGINT NULL,
    action VARCHAR(50) NOT NULL,
    quantity INTEGER NOT NULL,
    reason VARCHAR(255) NULL,
    created_at TIMESTAMPTZ NOT NULL
);
```

action 建議：

- `reserve`
- `release_reservation`
- `create_order`
- `payment_confirmed`
- `order_expired`
- `stock_reconcile`
- `compensation_rollback`

這張表適合 keyset pagination。

---

## 5. 敏感資料脫敏

後台 API 預設應該回傳脫敏資料。

範例：

```text
Email: user@example.com -> u***@example.com
姓名: 王小明 -> 王**
手機: 0912345678 -> 091****678
IP: 192.168.10.23 -> 192.168.*.*
```

目前 `users` 主要有 name / email，所以第一版先做：

- name masking
- email masking
- reveal API
- reveal audit log

不建議在作品中儲存信用卡號。付款卡號應由 payment provider 管理。

---

## 6. Reveal Sensitive API

完整敏感資料不應該直接出現在一般列表。

建議 API：

```http
POST /admin/orders/{orderId}/reveal-sensitive
```

request：

```json
{
  "reason": "客服核對訂單"
}
```

限制：

- 只有 `SUPER_ADMIN` 可用。
- `reason` 必填。
- 成功或失敗都可以寫 audit log。

response：

```json
{
  "order_id": 123,
  "user_id": 10,
  "user_name": "王小明",
  "email": "user@example.com"
}
```

---

## 7. Audit Log

建議新增 `admin_audit_logs`：

```sql
CREATE TABLE admin_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    admin_user_id BIGINT NOT NULL,
    action VARCHAR(100) NOT NULL,
    target_type VARCHAR(50) NOT NULL,
    target_id BIGINT NOT NULL,
    reason TEXT NULL,
    ip_address VARCHAR(64) NULL,
    user_agent TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL
);
```

第一版 action：

- `REVEAL_ORDER_SENSITIVE`
- `UPDATE_EVENT`
- `UPDATE_EVENT_SECTION`

後台操作紀錄也適合使用 keyset pagination。

---

## 8. 後台 API 第一版

建議第一版先做這些：

```http
GET  /admin/orders
GET  /admin/orders/{orderId}
POST /admin/orders/{orderId}/reveal-sensitive

GET  /admin/events
GET  /admin/events/{eventId}

GET  /admin/events/{eventId}/sections

GET  /admin/audit-logs
```

如果要先更小，可以只做：

```http
GET  /admin/orders
POST /admin/orders/{orderId}/reveal-sensitive
GET  /admin/audit-logs
```

---

## 9. Keyset Pagination 規格

適用列表：

- 訂單列表
- 搶票請求紀錄
- 庫存流水
- 後台操作紀錄

排序規則：

```sql
ORDER BY created_at DESC, id DESC
```

cursor 格式：

```json
{
  "created_at": "2026-06-28T12:00:00Z",
  "id": 12345
}
```

查下一頁：

```sql
WHERE created_at < :cursor_created_at
   OR (created_at = :cursor_created_at AND id < :cursor_id)
```

優點：

- 不會因為頁數越深越慢。
- 適合 append-only 或接近 append-only 的大表。
- 適合訂單、log、流水資料。

限制：

- 不適合任意跳頁。
- 排序欄位需要穩定。
- 前端要保存 `next_cursor`。

---

## 10. 建議實作順序

### Phase 1：後台基礎

1. 新增 admin role 模型。
2. 新增 admin auth middleware。
3. 新增 `admin_audit_logs` migration。
4. 新增 masking utility。

### Phase 2：訂單後台

1. 新增 `GET /admin/orders`。
2. 使用 keyset pagination。
3. 訂單列表回傳脫敏 user 資料。
4. 新增 `POST /admin/orders/{orderId}/reveal-sensitive`。
5. reveal 時寫入 audit log。

### Phase 3：活動與票區後台

1. 新增 `GET /admin/events`。
2. 新增 `GET /admin/events/{eventId}/sections`。
3. 顯示總庫存、鎖定中、已售、剩餘。
4. 後續再補活動與票區修改 API。

### Phase 4：操作紀錄

1. 新增 `GET /admin/audit-logs`。
2. 使用 keyset pagination。
3. IP 預設脫敏。

### Phase 5：進階紀錄

1. 新增搶票請求紀錄。
2. 新增庫存流水。
3. 將高頻紀錄改成 async 寫入。

---

## 11. README 可展示的亮點

後續可以在 README 放：

```text
Admin Backoffice
- Order list uses keyset pagination to avoid OFFSET deep pagination degradation.
- Sensitive user fields are masked by default in admin APIs.
- SUPER_ADMIN reveal requires a reason and writes audit logs.
- Event section inventory exposes total, reserved, sold, and available quantities.
- Stock ledger is planned for inventory reconciliation and compensation tracking.
```

---

## 12. 第一個可執行切片

最小可交付版本：

1. `admin_audit_logs` migration。
2. `MaskEmail()` / `MaskName()` utility。
3. `GET /admin/orders?limit=20`。
4. response 使用 `items + next_cursor`。
5. `POST /admin/orders/{orderId}/reveal-sensitive`。
6. reveal 寫入 audit log。

這個切片可以展示：

- 後台不是單純 CRUD。
- 大表查詢有 pagination 設計。
- 敏感資料有權限與留痕。
- 能直接接在目前購票主流程資料上。
