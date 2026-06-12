# Mock Payment Callback 測試流程

這份文件說明如何用 `ec-payment-service` 模擬第三方支付，回呼到 `buy-ticket`。

---

## 1. 前置設定

### `buy-ticket`

確認 `D:\SourceCode\Go\buy-ticket\config\local\app.properties` 至少有這三個設定：

```properties
payment.mock.merchant_id=TEST_MERCHANT
payment.mock.hash_key=TEST_SECRET
payment.mock.hash_iv=TEST_HASH_IV
```

啟動：

```bash
make run-local
```

預設 API：

```text
http://localhost:8080
```

callback endpoint：

```text
POST /payments/provider/ecpay/callback
```

完整 callback URL：

```text
http://localhost:8080/payments/provider/ecpay/callback
```

### `ec-payment-service`

它要把 callback 打到 `buy-ticket`，所以 callback URL 要設成：

```text
http://localhost:8080/payments/provider/ecpay/callback
```

如果 `ec-payment-service` 有自己的 merchant / hash 設定，必須和 `buy-ticket` 一致：

```text
MerchantID = TEST_MERCHANT
HashKey    = TEST_SECRET
HashIV     = TEST_HASH_IV
```

---

## 2. 建立一筆可付款訂單

先在 `buy-ticket` 內建立完整購票資料。

### Step 1: 查活動

```http
GET /events
```

### Step 2: 加入 queue

```http
POST /queue/join
Content-Type: application/json
```

```json
{
  "event_id": 1,
  "user_id": 1,
  "client_id": "postman-device-001",
  "request_id": "req-mock-pay-001",
  "channel": "web"
}
```

記下：

- `purchase_token`

### Step 3: 建 reservation

```http
POST /reservations
Content-Type: application/json
```

```json
{
  "user_id": 1,
  "event_id": 1,
  "section_id": 1,
  "quantity": 2,
  "hold_until": "2026-06-11T20:00:00+08:00",
  "purchase_token": "pt_xxx"
}
```

記下：

- `reservation.id`
- `total_amount`

### Step 4: 建 order

```http
POST /orders
Content-Type: application/json
```

```json
{
  "reservation_id": 1,
  "order_no": "ORD-MOCK-001",
  "expires_at": "2026-06-11T20:10:00+08:00"
}
```

記下：

- `order.id`
- `order_no`
- `total_amount`

---

## 3. 呼叫 mock payment service

對 `ec-payment-service` 發送付款請求：

```http
POST /api/payment/process
Content-Type: application/json
```

```json
{
  "recordNo": "ORD-MOCK-001",
  "amount": "5600",
  "payType": "ECPAY",
  "callbackUrl": "http://localhost:8080/payments/provider/ecpay/callback"
}
```

欄位對應：

- `recordNo` = `buy-ticket` 的 `order_no`
- `amount` = `order.total_amount`
- `callbackUrl` = `buy-ticket` callback endpoint

---

## 4. 預期結果

如果 callback 成功，`buy-ticket` 會：

- 驗 `CheckMacValue`
- 用 `MerchantTradeNo` 找到 order
- 確認 `RtnCode == 1`
- 呼叫 `PayOrder(...)`

之後你可以查：

```http
GET /orders/order-no/ORD-MOCK-001
GET /users/1/payments
```

預期：

- `order.status = paid`
- `reservation.status = confirmed`
- 新增一筆 payment
- `payment.payment_no = callback.TradeNo`

---

## 5. 失敗排查

### 簽章錯誤

如果看到類似：

```json
{
  "error": "invalid payment signature"
}
```

先檢查：

- `payment.mock.merchant_id`
- `payment.mock.hash_key`
- `payment.mock.hash_iv`
- `ec-payment-service` 的 MerchantID / HashKey / HashIV

這兩邊必須完全一致。

### 金額不一致

如果看到：

```json
{
  "error": "payment amount mismatch"
}
```

表示：

- `ec-payment-service` 的 `amount`
- `buy-ticket` 的 `order.total_amount`

不一致。

### 找不到訂單

如果看到 order not found 類錯誤，檢查：

- `recordNo`
- `order_no`

是否完全一致。

---

## 6. 最小驗證清單

每次串接 mock payment，至少驗這四件：

- `recordNo == order_no`
- `amount == order.total_amount`
- `callbackUrl` 指到 `buy-ticket`
- Merchant / HashKey / HashIV 兩邊一致
