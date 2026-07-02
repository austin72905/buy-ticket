# Payment Attempts 設計

`payment_attempts` 用來記錄每一次付款 provider 呼叫嘗試。

它不是最終付款結果表。最終付款結果仍然是 `payments`。

## 1. 為什麼需要

目前短期版付款流程是：

```text
POST /payments
後端直接把 order 改 paid
後端建立 payment
後端 confirmed reservation
```

之後要接 mock pay service 或正式 payment provider 時，不能一開始就把 order 改 paid。

更合理的流程是：

```text
POST /payments/start
建立 payment_attempt
呼叫 payment provider
等待 provider callback
callback 成功後才建立 payment
callback 成功後才把 order 改 paid
callback 成功後才 confirmed reservation
```

`payment_attempts` 就是中間那筆「付款嘗試」。

## 2. 和 payments 的差異

`payment_attempts`：

- 記錄每一次呼叫 provider 的嘗試。
- 可以是 processing、succeeded、failed、timeout、cancelled。
- 可以有多筆 attempts 對同一張 order。
- 適合記錄 provider、merchant_trade_no、request payload、callback payload、失敗原因。

`payments`：

- 記錄最終付款結果。
- 通常成功付款後才建立或標記 paid。
- 是 order paid 和 reservation confirmed 的結果依據。

## 3. 狀態

目前 domain 狀態：

```text
1 = processing
2 = succeeded
3 = failed
4 = timeout
5 = cancelled
```

## 4. 目前已完成

目前採用方案 A：

```text
POST /payments
= demo direct-pay
= 後端直接 paid
= 保留給開發與測試

POST /payments/start
= provider payment flow
= 建立 payment_attempt
= 不會直接把 order 改 paid
= 後續等待 mock pay service callback
```

目前已補上：

- migration：`000005_payment_attempts`
- domain：`PaymentAttempt`
- repository：`PaymentAttemptRepository`
- service：`CreatePaymentAttempt`
- API：`POST /payments/start`

目前還沒有改前端，但 `POST /payments/start` 已經會呼叫 mock pay service。

Mock pay service 預設設定：

```properties
payment.mock.base_url=http://localhost:8081
payment.mock.backup_base_url=
payment.mock.callback_url=http://localhost:8080/payments/provider/ecpay/callback
payment.mock.timeout_seconds=3
```

如果 mock pay service 沒有啟動，`POST /payments/start` 仍會建立 `payment_attempt`，但 attempt 會被標記為 `timeout`。

`POST /payments/start` request：

```http
POST /payments/start
Idempotency-Key: 018f0f58-8f62-7f5e-9f28-0c2a1b9c4e77
Content-Type: application/json
```

```json
{
  "order_id": 1,
  "method": "credit_card",
  "provider": "mock_ecpay"
}
```

Response：

```json
{
  "id": 1,
  "order_id": 1,
  "idempotency_key": "018f0f58-8f62-7f5e-9f28-0c2a1b9c4e77",
  "provider": "mock_ecpay",
  "merchant_trade_no": "ORD-1-ATT-20260618120000-000000000",
  "method": "credit_card",
  "amount": 2800,
  "status": 1,
  "created_at": "2026-06-18T12:00:00Z",
  "updated_at": "2026-06-18T12:00:00Z"
}
```

Status：

```text
202 Accepted
```

表示付款嘗試已建立，但付款尚未完成。

## 5. Provider router 與 circuit breaker

目前 provider-style 付款流程已加入 provider router：

```text
mock_ecpay_primary -> payment.mock.base_url
mock_ecpay_backup  -> payment.mock.backup_base_url
```

`payment.mock.backup_base_url` 預設為空，代表只使用 primary provider。設定 backup 後，新的 `payment_attempt` 可以在 primary breaker open 時改走 backup。

Circuit breaker 預設：

```properties
payment.breaker.enabled=true
payment.breaker.consecutive_failures=5
payment.breaker.open_timeout_seconds=30
payment.breaker.half_open_max_requests=1
```

行為規則：

- `4xx` provider response 視為 request 問題，不計入 breaker failure。
- `5xx`、connection refused、timeout 會計入 breaker failure。
- primary breaker open 時，新的 attempt 會選 backup。
- 既有 timeout attempt 不會跨 provider 重送。
- 同一個 `Idempotency-Key` retry 仍回同一筆 attempt。

範例：

```text
attempt 1:
  provider = mock_ecpay_primary
  status = timeout
  failure_reason = connection refused

attempt 2:
  provider = mock_ecpay_backup
  status = processing
```

這個設計的重點是：fallback 只套用在新的付款嘗試，不直接把舊 attempt 轉送到另一家 provider，避免重複扣款。

## 6. 下一步

後續可以接續做：

1. mock pay service callback 回 `POST /payments/provider/ecpay/callback`。
2. callback 用 `merchant_trade_no` 找回 `payment_attempt`。
3. callback 成功後，attempt 改 succeeded。
4. callback 成功後，建立 payment、order paid、reservation confirmed。
5. 前端改用 `POST /payments/start`。

## 7. 和 idempotency 的關係

`Idempotency-Key` 保護的是「前端到 buy-ticket」這次付款啟動請求。

`payment_attempts.merchant_trade_no` 保護的是「buy-ticket 到 payment provider」這次付款嘗試。

建議關係：

```text
Idempotency-Key
  -> payment_attempt
  -> merchant_trade_no
  -> provider callback
```

同一個 `Idempotency-Key` retry 時，應該回同一個 payment attempt，而不是建立新的 provider payment request。
