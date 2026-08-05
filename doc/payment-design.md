# Payment Design

## 目標

Payment 模組的目標是讓付款流程具備可靠性，而不是讓前端直接決定訂單是否付款成功。

目前設計重點：

- 前端使用 `POST /payments/start` 發起付款。
- 後端建立 `payment_attempt` 記錄每一次付款嘗試。
- 後端呼叫 mock payment provider。
- provider callback 回來後，後端驗證簽章與金額。
- callback 成功後，後端更新 payment / order / reservation。
- 若 callback 遺失或 timeout，scheduler 會用 reconciliation 主動查 provider。
- 付款成功後會寫入 outbox event，後續由 outbox publisher 發送通知。

## 核心資料表角色

### orders

代表使用者送出的訂單。

重要狀態：

- `PENDING_PAYMENT`
- `PAID`
- `EXPIRED`
- `CANCELLED`

### payments

代表最終付款結果。通常一筆成功付款會對應一筆 payment。

### payment_attempts

代表每一次對 provider 發起付款的嘗試。

它不是最終付款結果，而是 payment provider 互動紀錄。

用途：

- 記錄 provider name
- 記錄 merchant trade no
- 記錄 request / response / callback payload
- 記錄 timeout / failed / succeeded
- 支援 callback 關聯
- 支援 reconciliation 查詢

### idempotency_keys

用於 `POST /payments`，避免同一位使用者對同一 endpoint 的付款請求因 retry 被處理多次。

唯一範圍為 `user_id + key + endpoint`；不同使用者可以使用相同的 key。

`POST /payments/start` 的冪等紀錄則存放在 `payment_attempts.idempotency_key`，唯一範圍為
`order_id + idempotency_key`；不同訂單可以使用相同的 key。

## Payment Attempt 狀態

| Status | 意義 |
| --- | --- |
| `PROCESSING` | 已建立 attempt，等待 provider callback 或 reconciliation |
| `SUCCEEDED` | provider 確認付款成功 |
| `FAILED` | provider 明確回失敗 |
| `TIMEOUT` | 呼叫 provider timeout 或 circuit breaker open |

`TIMEOUT` 不代表一定付款失敗，只代表 buy-ticket 當下沒有拿到明確成功結果。

## 付款主流程

```text
frontend
  -> POST /payments/start
  -> buy-ticket create payment_attempt
  -> buy-ticket call mock payment provider
  -> mock provider callback buy-ticket
  -> buy-ticket verify callback
  -> buy-ticket mark payment_attempt succeeded
  -> buy-ticket create payment
  -> buy-ticket mark order paid
  -> buy-ticket confirm reservation
  -> buy-ticket create outbox event
```

前端不應該呼叫任何 API 直接把 order 改成 paid。

## POST /payments/start

用途：由前端發起付款流程。

Header：

```http
Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000
```

Request：

```json
{
  "order_id": 1
}
```

Response：

```json
{
  "payment_attempt_id": 10,
  "order_id": 1,
  "provider": "mock_ecpay_primary",
  "merchant_trade_no": "ORD-1-ATT-20260718091128-000000000",
  "status": "PROCESSING"
}
```

## Idempotency

`Idempotency-Key` 通常由前端產生，例如 UUID。

需要 idempotency 的原因：

### 第一次付款成功，但 response timeout

```text
frontend -> POST /payments/start
buy-ticket -> provider 成功
response 回 frontend 時 timeout
frontend retry
```

如果沒有 idempotency，retry 可能建立第二筆 payment attempt。

### 兩個 request 同時進來

使用者連點或網路重送時，兩個 request 可能同時進入後端。

idempotency 可以讓相同 key 的請求只被處理一次。

### provider timeout，但其實可能成功

provider timeout 不代表 provider 沒處理。這種情況不能直接換 provider 重打同一筆 attempt，避免重複扣款。

目前策略：

- 同一張訂單使用相同的 `Idempotency-Key` 時，回傳同一筆 payment attempt。
- 不同訂單可以使用相同的 `Idempotency-Key`。
- 付款失敗後若使用者要重新付款，應產生新的 `Idempotency-Key`。
- 新的付款請求會建立新的 `payment_attempt`。

### 併發請求處理

Idempotency 採用資料庫 unique constraint 作為最終一致性邊界，避免只依賴「先查詢、再新增」造成競爭：

```text
request A -> SELECT：找不到
request B -> SELECT：找不到
request A -> INSERT：成功
request B -> INSERT：unique violation
request B -> 重新查詢並讀回 request A 建立的紀錄
```

PostgreSQL 回傳 SQLSTATE `23505` 時，repository 會轉換為 unique constraint domain error，service 再依 idempotency scope 重新查詢：

- `POST /payments`：依 `user_id + key + endpoint` 讀回 `idempotency_keys`。
- `POST /payments/start`：依 `order_id + idempotency_key` 讀回 `payment_attempts`。
- 若既有紀錄的 request hash、付款方式或 provider 不相容，回傳 idempotency conflict。
- 若資料相容，回傳既有結果或 processing 狀態，不建立第二筆資料。

`POST /payments/start` 另外會區分 payment attempt 是由本次 request 新建，還是競爭後讀回。只有成功新建 attempt 的 request 可以呼叫 payment provider；讀回既有 attempt 的 request 會直接回傳該紀錄，避免重複送出付款請求。

## Provider Router 與 Circuit Breaker

目前支援 primary / backup mock payment provider。

設定：

```env
PAYMENT_MOCK_BASE_URL=http://localhost:8081
PAYMENT_MOCK_BACKUP_BASE_URL=
PAYMENT_BREAKER_ENABLED=true
PAYMENT_BREAKER_CONSECUTIVE_FAILURES=5
PAYMENT_BREAKER_OPEN_TIMEOUT_SECONDS=30
PAYMENT_BREAKER_HALF_OPEN_MAX_REQUESTS=1
```

行為：

- primary 可用時，新的 attempt 走 primary。
- primary breaker open 時，新的 attempt 可以走 backup。
- 舊的 timeout attempt 不會直接拿去 backup provider 重送。

這樣避免同一筆付款在不同 provider 被重複扣款。

## Mock Payment Callback

mock payment provider callback endpoint：

```http
POST /payments/provider/ecpay/callback
```

callback 會做：

1. 驗證簽章。
2. 找到對應 `payment_attempt`。
3. 驗證 amount 等於 order total amount。
4. 標記 attempt succeeded。
5. 建立 payment。
6. 將 order 改為 `PAID`。
7. 將 reservation 改為 confirmed。
8. 建立 outbox event。

callback 關聯以 `merchant_trade_no` 為主，不再依賴 `order_no` fallback。

## Payment Reconciliation

目的：補償 callback 遺失、callback 失敗、provider timeout 的情況。

scheduler 會定期查詢：

```text
status IN (PROCESSING, TIMEOUT)
created_at < now - delay
next_reconcile_at <= now
reconcile_attempts < max_attempts
```

預設策略：

```env
PAYMENT_RECONCILE_ENABLED=true
PAYMENT_RECONCILE_BATCH_SIZE=100
PAYMENT_RECONCILE_DELAY_SECONDS=120
PAYMENT_RECONCILE_RETRY_AFTER_SECONDS=30
PAYMENT_RECONCILE_MAX_ATTEMPTS=5
```

成功查到 provider 已付款時，會走和 callback 成功相同的完成付款流程。

如果查詢失敗或 provider 還未完成：

- `reconcile_attempts + 1`
- 設定 `next_reconcile_at`
- 記錄 `last_reconcile_error`

它不是一直狂查，而是有限次數、延遲查詢。

## Outbox Payment Notification

付款成功後會在同一個交易流程中建立 outbox event。

目的：

- 避免 order 已付款但通知事件漏掉。
- 讓通知發送可以非同步 retry。
- 未來可以接 RabbitMQ / Kafka / Email service。

目前 outbox publisher 先以 log 方式模擬發送。

設定：

```env
OUTBOX_PUBLISH_ENABLED=true
OUTBOX_PUBLISH_BATCH_SIZE=100
OUTBOX_PUBLISH_RETRY_AFTER_SECONDS=30
```

事件概念：

```json
{
  "event_id": "evt_0123456789abcdef",
  "event_type": "PAYMENT_SUCCEEDED",
  "aggregate_type": "PAYMENT",
  "aggregate_id": 1,
  "payload": {
    "payment_id": 1,
    "payment_no": "PAY-20260718171128-98e4ee",
    "order_id": 1,
    "order_no": "ORD-20260718170000-1234",
    "reservation_id": 1,
    "user_id": 1,
    "event_id": 1,
    "section_id": 1,
    "quantity": 2,
    "amount": 2800,
    "method": "ecpay",
    "paid_at": "2026-07-18T17:11:28+08:00"
  }
}
```

`aggregate_id` 使用 `payment.ID`，不是 order ID。每個 outbox event 的 `event_id` 具有唯一索引，publisher 失敗時最多重試 5 次並透過 `next_attempt_at` 延後下一次嘗試。

## 為什麼不是前端直接改 paid

付款成功與否必須由後端決定，原因：

- 後端需要查 order amount。
- 後端需要驗證 provider callback 簽章。
- 後端使用 server time 決定 paid time。
- 後端需要一致更新 payment / order / reservation。
- 前端資料不可被信任。

## 常見錯誤

### invalid payment signature

通常代表：

- `PAYMENT_MOCK_MERCHANT_ID`
- `PAYMENT_MOCK_HASH_KEY`
- `PAYMENT_MOCK_HASH_IV`

和 mock payment service 設定不一致。

### payment amount mismatch

代表 provider callback amount 和 order total amount 不一致。這種情況不可標記 paid。

### callback 後 order 沒變 paid

常見原因：

- callback 沒打到 buy-ticket。
- callback 簽章錯誤。
- 找不到 merchant trade no 對應 attempt。
- amount mismatch。
- order 已過期或狀態不允許更新。

## 限制與後續補強

目前限制：

- mock payment provider 不是真實金流。
- outbox publisher 目前以 log 模擬，不是真的送 MQ / Email。
- circuit breaker 狀態是 process-local，多 Pod 不共享。
- payment reconciliation 依賴 provider query API。
- outbox publish 與 payment reconciliation 尚未使用 claim／row locking；目前應維持單一 scheduler replica。

後續可補強：

- 接真實 MQ。
- 補 Email notification service。
- payment provider 增加正式查詢 API 與簽章驗證。
- circuit breaker metrics / dashboard。
- 多 scheduler worker 前，替 outbox publish 與 payment reconciliation 加入 claim／locking，並讓訊息 consumer 依 `event_id` 做冪等處理。
