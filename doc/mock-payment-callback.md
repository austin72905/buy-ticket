# Mock Payment Callback 流程

這份文件記錄 `buy-ticket` 如何接 `ec-payment-service` mock payment provider。

目前正式方向是：

```text
frontend
  -> POST /payments/start
  -> buy-ticket 建立 payment_attempt
  -> buy-ticket 呼叫 ec-payment-service
  -> ec-payment-service callback buy-ticket
  -> buy-ticket 建立 payment、更新 order paid、更新 reservation confirmed
```

舊的直接付款 API `POST /payments` 仍保留作為 demo direct-pay，但前端付款頁已改用 `POST /payments/start`。

---

## 1. 啟動條件

### buy-ticket

建議用 `dev` 環境，因為 `dev` 會使用 Postgres 與 Redis：

```powershell
make run-dev
```

`config/dev/app.properties` 需要有：

```properties
payment.mock.merchant_id=TEST_MERCHANT
payment.mock.hash_key=TEST_SECRET
payment.mock.hash_iv=TEST_HASH_IV
payment.mock.base_url=http://localhost:8081
payment.mock.backup_base_url=
payment.mock.callback_url=http://localhost:8080/payments/provider/ecpay/callback
payment.mock.timeout_seconds=3
payment.breaker.enabled=true
payment.breaker.consecutive_failures=5
payment.breaker.open_timeout_seconds=30
payment.breaker.half_open_max_requests=1
```

### Payment provider router 與 circuit breaker

目前 `POST /payments/start` 會透過 provider router 選擇 mock payment provider：

```text
mock_ecpay_primary -> payment.mock.base_url
mock_ecpay_backup  -> payment.mock.backup_base_url
```

`payment.mock.backup_base_url` 預設為空，代表只使用 primary provider。若要啟用 backup：

```properties
payment.mock.backup_base_url=http://localhost:8082
```

或用環境變數：

```powershell
$env:PAYMENT_MOCK_BACKUP_BASE_URL="http://localhost:8082"
```

Circuit breaker 預設策略：

```text
連續 5 次 provider 呼叫失敗 -> breaker open
open 30 秒 -> half-open
half-open 只允許 1 次探測請求
```

重要規則：

- breaker 只保護 provider 呼叫，不會把既有 `payment_attempt` 跨 provider 重送。
- 若 primary breaker open，新的 `payment_attempt` 才會選 backup。
- 舊的 timeout attempt 保留原 provider 與狀態，等待 callback 或後續 reconciliation。
- 這樣可以避免 primary 其實已成功但 response timeout 時，又自動送到 backup 造成重複扣款。

buy-ticket API：

```text
http://localhost:8080
```

callback endpoint：

```text
POST http://localhost:8080/payments/provider/ecpay/callback
```

### ec-payment-service

mock pay service 預期啟動在：

```text
http://localhost:8081
```

健康檢查：

```powershell
curl http://localhost:8081/health
```

正常回應範例：

```json
{
  "status": "ok",
  "service": "payment-service"
}
```

`ec-payment-service` 的 merchant / hash 設定需要與 `buy-ticket` 一致：

```text
MerchantID = TEST_MERCHANT
HashKey    = TEST_SECRET
HashIV     = TEST_HASH_IV
```

---

## 2. 正式付款流程

### Step 1：完成購票前置流程

前端或 Postman 先完成：

1. `POST /auth/login`
2. `POST /queue/join`
3. `GET /queue/status/{queueToken}` 取得 `purchase_token`
4. `POST /reservations`
5. `POST /orders`

完成後會得到一筆 `pending_payment` order。

### Step 2：前端啟動付款

前端呼叫：

```http
POST /payments/start
Idempotency-Key: 018f0f58-8f62-7f5e-9f28-0c2a1b9c4e77
Content-Type: application/json
```

Request：

```json
{
  "order_id": 1,
  "method": "credit_card",
  "provider": "mock_ecpay"
}
```

後端會做：

1. 查詢 order。
2. 確認 order 狀態是 `pending_payment`。
3. 用 `order.total_amount` 決定付款金額。
4. 建立 `payment_attempts`。
5. 產生 `merchant_trade_no`。
6. 呼叫 mock pay service。

Response：

```http
202 Accepted
```

```json
{
  "id": 1,
  "order_id": 1,
  "idempotency_key": "018f0f58-8f62-7f5e-9f28-0c2a1b9c4e77",
  "provider": "mock_ecpay",
  "merchant_trade_no": "MT-1-20260622120000",
  "method": "credit_card",
  "amount": 2800,
  "status": 1,
  "created_at": "2026-06-22T12:00:00Z",
  "updated_at": "2026-06-22T12:00:00Z"
}
```

狀態說明：

```text
1 = processing
2 = succeeded
3 = failed
4 = timeout
5 = cancelled
```

### Step 3：buy-ticket 呼叫 mock pay service

`buy-ticket` 會自動呼叫：

```http
POST http://localhost:8081/api/payment/process
Content-Type: application/json
```

Request payload：

```json
{
  "recordNo": "MT-1-20260622120000",
  "amount": "2800",
  "payType": "ECPAY",
  "callbackUrl": "http://localhost:8080/payments/provider/ecpay/callback"
}
```

欄位對應：

- `recordNo` = `payment_attempts.merchant_trade_no`
- `amount` = `orders.total_amount`
- `payType` = 目前固定送 `ECPAY`
- `callbackUrl` = `payment.mock.callback_url`

重點：`recordNo` 不是 `order_no`。它是這一次付款 attempt 的交易編號。

---

## 3. Callback 處理

mock pay service 成功處理後，會 callback：

```http
POST /payments/provider/ecpay/callback
Content-Type: application/x-www-form-urlencoded
```

主要欄位：

```text
MerchantID
MerchantTradeNo
RtnCode
RtnMsg
TradeNo
TradeAmt
PaymentDate
PaymentType
CheckMacValue
```

callback handler 會做：

1. 驗證 `CheckMacValue`。
2. 檢查 `RtnCode`。
3. 用 `MerchantTradeNo` 查 `payment_attempts.merchant_trade_no`。
4. 用 `payment_attempts.order_id` 查 order。
5. 檢查 `TradeAmt` 是否等於 `order.total_amount`。
6. 建立 `payments`。
7. 更新 order 為 `paid`。
8. 更新 reservation 為 `confirmed`。
9. 更新 payment attempt 為 `succeeded`。

成功後可查：

```http
GET /orders/{orderId}
GET /me/payments
```

預期結果：

- `orders.status = 2`，代表 paid。
- `reservations.status = 2`，代表 confirmed。
- 新增一筆 `payments`。
- `payments.payment_no = callback.TradeNo`。
- `payment_attempts.status = 2`。
- `payment_attempts.provider_trade_no = callback.TradeNo`。
- `payment_attempts.payment_id` 指向新增的 payment。

---

## 4. Timeout 與 provider 未啟動

如果 `ec-payment-service` 沒有啟動，或 `POST /api/payment/process` 超時：

- `/payments/start` 仍會建立 `payment_attempts`。
- attempt 會被標記為 `timeout`。
- order 不會變 paid。
- reservation 不會變 confirmed。

這是刻意設計，避免 payment provider 不穩定時讓主購票流程直接崩潰。

---

## 5. Provider timeout、breaker 與重啟付款

如果 `ec-payment-service` 沒有啟動，或 `POST /api/payment/process` 超時：

- `/payments/start` 仍會建立 `payment_attempts`。
- attempt 會被標記為 `timeout`。
- order 不會改成 paid。
- reservation 不會改成 confirmed。

如果 primary provider 已觸發 circuit breaker：

- 新的付款啟動請求會優先跳過 primary。
- 若有設定 backup provider，新的 `payment_attempt` 會記錄 `provider = mock_ecpay_backup`。
- 若沒有可用 provider，仍會建立 `payment_attempt`，但狀態會是 `timeout`，`failure_reason` 會記錄 circuit breaker open。
- 前端可以讓使用者按「Restart Mock Payment」建立新的 attempt；這會使用新的 `Idempotency-Key`。

同一個 `Idempotency-Key` retry 時仍會回同一筆 attempt，不會因為 retry 就切換 provider。

---

## 6. Idempotency 行為

前端呼叫 `POST /payments/start` 時應送：

```http
Idempotency-Key: <uuid>
```

用途：

- 避免使用者重按付款按鈕建立多筆 provider payment attempt。
- 避免前端 timeout 後 retry 造成重複付款。
- 同一個 key、同一個 request 會回同一筆 attempt。
- 同一個 key、不同 request 會回 idempotency conflict。

---

## 7. Callback 關聯規則

目前 callback 已不再支援舊的 `MerchantTradeNo -> order_no` fallback。

callback 必須用：

```text
MerchantTradeNo -> payment_attempts.merchant_trade_no -> payment_attempts.order_id -> orders.id
```

如果找不到 `payment_attempts`，callback 會失敗，不會改 order，也不會建立 payment。

---

## 8. 常見問題

### invalid payment signature

代表 `CheckMacValue` 驗證失敗。

檢查：

- `payment.mock.merchant_id`
- `payment.mock.hash_key`
- `payment.mock.hash_iv`
- `ec-payment-service` 的 MerchantID / HashKey / HashIV

兩邊必須一致。

### payment amount mismatch

代表 callback 的 `TradeAmt` 與 `order.total_amount` 不一致。

檢查：

- `payment_attempts.amount`
- `orders.total_amount`
- mock pay service callback 回來的 `TradeAmt`

### callback 後 order 沒變 paid

檢查：

1. `payment_attempts.merchant_trade_no` 是否等於 callback 的 `MerchantTradeNo`。
2. `payment_attempts.status` 是否仍是 `processing` 或 `timeout`。
3. callback 的 `RtnCode` 是否為 `1`。
4. callback 的 `CheckMacValue` 是否正確。
5. `TradeAmt` 是否等於 `orders.total_amount`。

### /payments/start 回 timeout

代表 `buy-ticket` 有建立 attempt，但呼叫 mock pay service 失敗。

檢查：

```powershell
curl http://localhost:8081/health
```

以及：

```properties
payment.mock.base_url=http://localhost:8081
payment.mock.backup_base_url=http://localhost:8082
payment.mock.callback_url=http://localhost:8080/payments/provider/ecpay/callback
```
