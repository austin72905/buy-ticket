# Queue / Purchase Token 設計

本文整理 queue 與 purchase token 的責任切分，對齊目前實作。

---

## 1. 目標

這套設計要解決幾件事：

- 大量使用者同時搶票時，先進 queue，不直接打購票流程
- 系統控制放行節奏，不讓所有人同時進入庫存扣減
- 被放行的人拿到短效、一次性的 `purchase_token`
- `purchase_token` 只能用來建立 reservation

---

## 2. 流程

1. `GET /sale/status`
2. `POST /queue/join`
3. `GET /queue/status/{queueToken}`
4. 背景 job 放行 queue
5. 取得 `purchase_token`
6. `POST /reservations`
7. `POST /orders`
8. `POST /payments`

---

## 3. Queue status

queue status 使用 `int enum`：

| 值 | 名稱 | 說明 |
|---|---|---|
| `1` | `waiting` | 排隊中 |
| `2` | `ready` | 已放行，可建立 reservation |
| `3` | `expired` | queue token 已失效 |

目前刻意只保留最小狀態集，不額外拆 `cancelled`、`rejected`。

---

## 4. Queue 的責任

queue 負責：

- 接住高流量使用者
- 記錄排隊順序
- 提供目前位置與前方人數
- 配合背景 job 決定何時放行

queue 不負責：

- 直接扣票
- 直接建立 order
- 直接確認付款

---

## 5. Purchase token 的責任

purchase token 是 queue 放行後發出的短效憑證。

### 作用

- 證明此使用者已被 queue 放行
- 允許呼叫 `POST /reservations`
- 避免未排隊的人直接保留票券

### 特性

- 短效
- 一次性
- 綁定 `user_id`
- 綁定 `event_id`

### 使用時機

- 只在 `POST /reservations` 消耗

---

## 6. 為什麼先 queue，再 reservation

這樣分層是有必要的：

- queue 解決入口流量控制
- reservation 解決票券暫時保留
- order 解決交易單據
- payment 解決付款結果

如果 queue 放行後直接建 order，會讓：

- order 膨脹太快
- 未付款超時清理壓力變大
- domain 邊界變混亂

所以目前分成：

- queue / purchase token
- reservation
- order
- payment

是合理的。

---

## 7. 背景 job

### Queue 放行 job

- job name: `queue-promote-ready`
- cron: `*/1 * * * * *`
- 作用：把 waiting 使用者推進 ready

### Order 過期 job

- job name: `order-expire-sweep`
- cron: `*/5 * * * * *`
- 作用：掃描超時未付款訂單並釋放資源

---

## 8. Redis 結構

### queue 排序

- `queue:event:{eventId}`
  - sorted set
  - score = sequence
  - member = queue token

### sequence

- `queue:event:{eventId}:seq`
  - 產生遞增序號

### queue snapshot

- `queue:token:{queueToken}`

### purchase token mapping

- `queue:purchase:{purchaseToken}`

### user / event 去重

- `queue:user:event:{eventId}:{userId}`

### active events

- `queue:events:active`

---

## 9. Reservation 階段

目前流程是：

1. 使用者排隊
2. 被放行
3. 拿到 `purchase_token`
4. 呼叫 `POST /reservations`
5. `BookingService.ReserveTicket(...)` 先消耗 `purchase_token`
6. reservation 建立成功後，才允許後續 order / payment

這代表 queue 並不直接保票，只負責放行資格。

---

## 10. 後續可以再補

- queue timeout cleanup
- purchase token cleanup
- Redis + Lua 原子扣庫存
- payment callback
- RabbitMQ delay / DLQ timeout

---

## 11. 為什麼 purchase token 不能只靠 TTL

`purchase_token` 雖然有 TTL，但它只會讓單一 key 過期，不會自動清掉整個 queue 關聯。

目前 `purchase_token` 過期後，還可能殘留：

- queue snapshot 的 `ready` 狀態
- `queue:user:event:{eventId}:{userId}` 索引
- `queue:event:{eventId}` sorted set member

如果這些資料不清：

- queue rank 可能不準
- 使用者可能被舊 token 卡住
- 後續放行與重排隊會出現髒資料

所以需要額外的 `purchase token cleanup` job，負責把過期 token 對應的 queue 關聯一起清乾淨。

---

## 12. 為什麼 queue timeout 也要 cleanup

queue token 到期後，也不能只靠 TTL。

原因和 purchase token 類似：

- sorted set member 不會自動消失
- user/event 索引不會自動一起清
- active event 標記也可能殘留

所以仍然需要 `queue timeout cleanup`，負責把整個 queue 關聯完整收尾。
