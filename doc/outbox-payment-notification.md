# Outbox Payment Notification

這份文件說明付款成功通知目前如何使用 Transactional Outbox Pattern。

## 目的

付款成功後除了更新核心交易狀態，也會產生非同步副作用：

```text
通知使用者付款成功
更新報表或投影
未來發送 RabbitMQ / Kafka event
```

這些副作用不應該卡住 payment callback，也不應該因為通知失敗導致訂單付款失敗。

## 目前流程

付款成功時：

```text
provider callback / payment reconciliation
-> PayOrder()
-> 同一個 DB transaction 內：
   1. 建立 payments
   2. orders 改 PAID
   3. reservations 改 CONFIRMED
   4. event_sections reserved -> sold
   5. 寫入 outbox_events: PAYMENT_SUCCEEDED
```

transaction commit 成功後，scheduler 會處理 outbox：

```text
outbox-publish
-> 掃 pending outbox_events
-> 第一版先 log 模擬付款成功通知
-> 成功後標記 PUBLISHED
-> 失敗則 attempts + 1，排下次 retry
```

## Table

```text
outbox_events
- event_id: 外部事件識別碼
- event_type: 例如 PAYMENT_SUCCEEDED
- aggregate_type: 例如 PAYMENT
- aggregate_id: payment id
- payload: event payload JSON
- status: PENDING / PUBLISHED / FAILED
- attempts: 已嘗試發布次數
- max_attempts: 最大嘗試次數
- next_attempt_at: 下次可嘗試時間
- last_error: 最後一次錯誤
- published_at: 發布成功時間
```

## Payload

`PAYMENT_SUCCEEDED` payload：

```json
{
  "payment_id": 1,
  "payment_no": "TRADE-001",
  "order_id": 20,
  "order_no": "ORD-001",
  "reservation_id": 10,
  "user_id": 3,
  "event_id": 1,
  "section_id": 2,
  "quantity": 2,
  "amount": 3600,
  "method": "ecpay_credit",
  "paid_at": "2026-07-18T12:00:00+08:00"
}
```

## 設定

```properties
outbox.publish.enabled=true
outbox.publish.batch_size=100
outbox.publish.retry_after_seconds=30
```

Helm 也有對應設定：

```yaml
outbox:
  publish:
    enabled: true
    batchSize: 100
    retryAfterSeconds: 30
```

## 為什麼目前先 log

第一版先不接 SMTP / SendGrid / SES，原因：

```text
避免引入外部金鑰與寄信成本
先展示可靠事件記錄與 retry 流程
未來可把 publishOutboxEvent 換成 RabbitMQ / Kafka / mail service
```

這樣可以先證明：

```text
核心付款交易成功
事件一定會被寫入 DB
通知失敗可重試
副作用不影響付款 callback
```
