# Payment Reconciliation

這份文件說明 `buy-ticket` 在 payment callback 遺失時，如何用排程主動查詢支付服務狀態並補償入帳。

## 目的

正常付款主流程仍然是 provider callback：

```text
POST /payments/start
-> buy-ticket 建立 payment_attempt = PROCESSING
-> buy-ticket 呼叫 mock payment service
-> mock payment service callback buy-ticket
-> buy-ticket 建立 payment、order 改 PAID、reservation 改 CONFIRMED
```

reconciliation 只處理異常情境：

```text
mock payment service 已經付款成功
但 callback 連續重試後仍失敗
buy-ticket 的 order 還停在 PENDING_PAYMENT
```

## 查詢策略

排程只查「可疑的 payment attempt」，不掃全部資料。

條件：

```sql
status IN (PROCESSING, TIMEOUT)
created_at <= now - payment.reconcile.delay_seconds
next_reconcile_at IS NULL OR next_reconcile_at <= now
reconcile_attempts < payment.reconcile.max_attempts
```

預設：

```properties
payment.reconcile.enabled=true
payment.reconcile.batch_size=100
payment.reconcile.delay_seconds=120
payment.reconcile.retry_after_seconds=30
payment.reconcile.max_attempts=5
```

排程：

```text
payment-attempt-reconcile
每 30 秒執行一次
每次最多查 100 筆
```

## Provider Query

目前 HTTP mock payment client 預設查詢：

```http
GET /api/payment/query?recordNo={merchant_trade_no}&merchantTradeNo={merchant_trade_no}
```

回傳格式固定為：

```json
{
  "merchant_trade_no": "MT-xxx",
  "provider_trade_no": "TRADE-xxx",
  "status": "SUCCESS",
  "amount": 2800,
  "paid_at": "2026-07-18T12:00:00+08:00",
  "method": "Credit",
  "failure_reason": ""
}
```

`status` 固定使用：
```text
SUCCESS -> 補入帳
FAILED -> attempt 標記 failed
PENDING -> 記錄錯誤並排下次查詢
```

## 成功補償會做什麼

查到付款成功後，會走與 callback 相同的入帳邏輯：

```text
建立 payment
order 改 PAID
reservation 改 CONFIRMED
section reserved -> sold
payment_attempt 改 SUCCEEDED
```

同一筆 provider trade number 已經建立過 payment 時，不會重複建立 payment。

## 為什麼不是一直查

callback 是主路徑，reconciliation 是補償路徑。

不直接一直查全部 payment_attempt 的原因：

```text
避免打爆 payment service
避免 payment service 故障時造成查詢風暴
避免大量 DB 掃描
正常情況 callback 已經能快速完成付款狀態更新
```
