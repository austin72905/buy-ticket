# buy-ticket Roadmap

本文整理目前 `buy-ticket` 專案的現況、已完成內容與下一步建議。

---

## 1. 目前目標

這個專案目前的主軸是先做出一條可運作的搶票核心流程：

- 活動查詢
- queue / waiting room
- purchase token
- reservation
- order
- payment
- order timeout cleanup

現階段優先順序是：

1. 先把核心交易流程做完整
2. 再把高併發控制補強
3. 最後補事件驅動與非同步機制

---

## 2. 已完成

### 2.1 Domain

目前已建立主要 domain：

- `Event`
- `Section`
- `Reservation`
- `Order`
- `Payment`

目前已有基本業務方法，例如：

- 活動是否可售
- section reserve / release
- reservation expire / cancel
- order pay / expire / cancel

### 2.2 Service

目前 `BookingService` 已有：

- `ReserveTicket`
- `CreateOrder`
- `PayOrder`
- `ExpireOrder`
- `ExpireReservation`
- `CancelReservation`
- `JoinQueue`
- `GetQueueStatus`
- `GetSaleStatus`
- `SweepExpiredOrders`

### 2.3 Controller / API

目前已補上的 API 包含：

- `GET /healthz`
- `GET /sale/status`
- `GET /events`
- `GET /events/:eventId`
- `GET /events/:eventId/sections`
- `GET /events/:eventId/availability`
- `GET /users/:userId/reservations`
- `GET /users/:userId/orders`
- `GET /users/:userId/payments`
- `GET /reservations/:reservationId`
- `GET /orders/:orderId`
- `GET /orders/order-no/:orderNo`
- `GET /payments/:paymentNo`
- `GET /queue/status/:queueToken`
- `POST /queue/join`
- `POST /reservations`
- `POST /orders`
- `POST /payments`
- `POST /reservations/expire`
- `POST /reservations/cancel`

### 2.4 Persistence

目前已完成：

- PostgreSQL migration
- `sqlc.yaml`
- `db/query.sql`
- memory repository
- postgres repository
- seed data

### 2.5 Queue / Redis

目前 queue 已支援：

- memory queue store
- Redis queue store
- sorted set 排隊順序
- queue token
- purchase token
- 使用者重複進 queue 去重
- 背景 job 放行 ready

### 2.6 Background jobs

目前已補上兩個背景 job：

- `queue-promote-ready`
  - cron: `*/1 * * * * *`
- `order-expire-sweep`
  - cron: `*/5 * * * * *`

目前排程直接寫在 `main.go`，不從 properties 讀取。

---

## 3. 目前技術設計

### 3.1 Queue 狀態

queue status 目前收斂成最小集合：

- `1` = waiting
- `2` = ready
- `3` = expired

不額外拆 `cancelled`、`rejected`。

### 3.2 Queue 與 Purchase Token

目前設計是：

1. 使用者先進 queue
2. 背景 job 控制放行
3. 被放行後拿到 `purchase_token`
4. 用 `purchase_token` 建 reservation
5. reservation 成功後再建立 order
6. payment 成功後完成交易

這樣可以把：

- 流量控制
- 暫時保留票券
- 訂單交易
- 付款結果

拆成清楚的階段。

### 3.3 Background job 控制

目前背景 job 的頻率固定寫在程式內：

- queue 放行：每 1 秒
- order expire sweep：每 5 秒

目前仍保留這些可配置參數：

- `queue.release.limit`
- `order.expire.batch.size`

---

## 4. 現在最值得做的下一步

### Phase 1：補齊庫存扣減策略

建議先補：

- Redis + Lua 原子扣庫存
- reservation 與庫存一致性

原因：

- 目前 queue 已經有了
- order / payment 也有了
- 真正高併發下，下一個核心風險就是庫存扣減

### Phase 2：補 timeout / cleanup

可以再補：

- queue timeout cleanup
- purchase token cleanup

### Phase 3：補事件驅動

之後再考慮：

- RabbitMQ delay / DLQ timeout
- payment callback
- 非同步事件通知

---

## 5. 目前不急著做的

這些不是不能做，而是現在優先度沒那麼高：

- gRPC
- 過度拆分 service
- 複雜 workflow engine
- 過早導入太多 MQ 流程

現階段先把單體流程做穩，比提早分散式化更重要。

---

## 6. 文件索引

目前建議先看這幾份文件：

- `D:\SourceCode\Go\buy-ticket\doc\api-flow.md`
- `D:\SourceCode\Go\buy-ticket\doc\queue-api.md`
- `D:\SourceCode\Go\buy-ticket\doc\queue-purchase-token-design.md`
- `D:\SourceCode\Go\buy-ticket\doc\encoding.md`

---

## 7. 開發注意事項

- 中文文件與程式碼請統一使用 `UTF-8`
- 優先使用 `apply_patch` 修改 `.go`、`.md`、`.yaml`、`.json`、`.sql`
- 避免使用 PowerShell 的 `Get-Content | Set-Content` 直接覆蓋中文檔
- 如果一定要用 PowerShell 重寫文字檔，請明確指定 UTF-8 No BOM

---

## 8. 總結

目前專案已經不是空殼，已經有一條可跑的主流程：

- sale status
- join queue
- queue status
- reservation
- order
- payment
- order expire cleanup

下一步最值得投資的是：

- 庫存一致性
- Redis / Lua 原子扣減
- timeout 補強
