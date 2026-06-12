# Queue API Contract

本文整理目前 queue 相關 API 與行為，內容對齊現在專案實作。

---

## 1. 範圍

目前 queue 流程包含：

- `POST /queue/join`
- `GET /queue/status/{queueToken}`
- queue background job
- purchase token 放行

---

## 2. Queue status enum

`status` 使用 `int`，不使用 string。

| 值 | 名稱 | 說明 |
|---|---|---|
| `1` | `waiting` | 還在排隊中 |
| `2` | `ready` | 已放行，可取得 `purchase_token` |
| `3` | `expired` | queue token 已失效 |

---

## 3. Queue worker 行為

目前 queue store 不在查詢 API 當下自動放行，而是由背景 job 控制。

### 規則

- 背景 job 每 1 秒執行一次
- 呼叫 `PromoteReady(...)`
- 最多放行到 `queue.release.limit`
- 使用者被放行後，`status` 變成 `ready`
- `ready` 狀態會附帶 `purchase_token`

### 目前設定

```properties
queue.store=redis
queue.release.limit=1
```

補充：

- 放行頻率目前寫死在程式內：每 1 秒一次
- `queue.release.limit` 仍可透過 properties 調整

---

## 4. POST /queue/join

### 用途

- 建立 queue token
- 讓使用者進入排隊
- 若當前可直接放行，也可能直接拿到 `ready`

### Request

```json
{
  "event_id": 1,
  "user_id": 1,
  "client_id": "web-device-001",
  "request_id": "req-20260610-0001",
  "channel": "web",
  "access_code": ""
}
```

### 欄位說明

- `event_id`: 活動 ID
- `user_id`: 使用者 ID
- `client_id`: 裝置或 session 識別
- `request_id`: 去重用 request id
- `channel`: 來源，例如 `web` / `app`
- `access_code`: 特殊活動可用的進場碼

### Response: waiting

```json
{
  "queue_token": "qt_01JX1234567890ABCDEFG",
  "status": 1,
  "event_id": 1,
  "user_id": 1,
  "queue_position": 153,
  "ahead_count": 152,
  "estimated_wait_seconds": 4560,
  "joined_at": "2026-06-10T20:00:00+08:00",
  "expired_at": "2026-06-10T20:30:00+08:00"
}
```

### Response: ready

```json
{
  "queue_token": "qt_01JX1234567890ABCDEFG",
  "status": 2,
  "event_id": 1,
  "user_id": 1,
  "queue_position": 1,
  "ahead_count": 0,
  "estimated_wait_seconds": 0,
  "purchase_token": "pt_01JXABCDEFG1234567890",
  "purchase_token_expires_at": "2026-06-10T20:05:00+08:00",
  "joined_at": "2026-06-10T20:00:00+08:00",
  "expired_at": "2026-06-10T20:30:00+08:00"
}
```

---

## 5. GET /queue/status/{queueToken}

### 用途

- 查目前 queue 狀態
- 查目前前面還有幾個人
- 查是否已取得 `purchase_token`

### Response: waiting

```json
{
  "queue_token": "qt_01JX1234567890ABCDEFG",
  "status": 1,
  "event_id": 1,
  "user_id": 1,
  "queue_position": 88,
  "ahead_count": 87,
  "estimated_wait_seconds": 2610,
  "joined_at": "2026-06-10T20:00:00+08:00",
  "expired_at": "2026-06-10T20:30:00+08:00",
  "updated_at": "2026-06-10T20:02:10+08:00"
}
```

### Response: ready

```json
{
  "queue_token": "qt_01JX1234567890ABCDEFG",
  "status": 2,
  "event_id": 1,
  "user_id": 1,
  "queue_position": 1,
  "ahead_count": 0,
  "estimated_wait_seconds": 0,
  "purchase_token": "pt_01JXABCDEFG1234567890",
  "purchase_token_expires_at": "2026-06-10T20:05:00+08:00",
  "joined_at": "2026-06-10T20:00:00+08:00",
  "expired_at": "2026-06-10T20:30:00+08:00",
  "updated_at": "2026-06-10T20:03:00+08:00"
}
```

---

## 6. 錯誤案例

### 活動不可售

```json
{
  "error": "event is not on sale"
}
```

### 已經排過隊

```json
{
  "error": "user already joined queue"
}
```

### queue token 不存在

```json
{
  "error": "queue token not found"
}
```

---

## 7. Redis key 設計

目前 Redis queue store 使用以下 key：

- `queue:event:{eventId}`
  - `sorted set`
  - score = `INCR` sequence
  - member = `queue_token`
- `queue:event:{eventId}:seq`
  - event 專用 sequence
- `queue:token:{queueToken}`
  - queue snapshot
- `queue:purchase:{purchaseToken}`
  - purchase token -> queue token
- `queue:user:event:{eventId}:{userId}`
  - user/event -> queue token
- `queue:events:active`
  - 目前有 queue 的 event 集合

---

## 8. 串接順序

1. `GET /sale/status`
2. `POST /queue/join`
3. `GET /queue/status/{queueToken}`
4. 取得 `purchase_token`
5. `POST /reservations`
6. `POST /orders`
7. `POST /payments`

---

## 9. 為什麼需要 purchase token cleanup

即使 `purchase_token` 本身有 Redis TTL，仍然需要額外的 cleanup job。

原因是目前 queue 結構不是只有單一 key，而是多個關聯資料一起運作：

- `queue:purchase:{purchaseToken}`
- `queue:token:{queueToken}`
- `queue:user:event:{eventId}:{userId}`
- `queue:event:{eventId}` sorted set
- `queue:events:active`

如果只有 TTL：

- `queue:purchase:{purchaseToken}` 會自己過期
- 但 `queue:event:{eventId}` 裡的 member 不會自動移除
- `queue:user:event:{eventId}:{userId}` 也可能殘留
- queue snapshot 仍可能保持 `ready`

這會造成幾個問題：

- 前方排隊人數計算不準
- 使用者可能被舊 queue token 卡住，不能重新排隊
- sorted set 內殘留過期 token，形成幽靈資料

所以 cleanup job 的責任是把「過期 purchase token 對應的整組關聯」一起清掉。

### 目前 cleanup job 會做的事

- 找出 `ready` 且 `purchase_token` 已過期的 queue snapshot
- 刪除 `queue:purchase:{purchaseToken}`
- 刪除 `queue:user:event:{eventId}:{userId}`
- 從 `queue:event:{eventId}` 移除該 `queueToken`
- 將 queue snapshot 狀態改成 `expired`

### 結論

TTL 是第一層保護，用來讓單一 key 自動死亡。  
cleanup job 是第二層保護，用來維持多 key 關聯資料的一致性。

---

## 10. 為什麼還需要 queue timeout cleanup

`queue_token` 本身也有 TTL，但 queue 結構同樣不是只有單一 key。

如果只有 TTL，還可能殘留：

- `queue:event:{eventId}` sorted set member
- `queue:user:event:{eventId}:{userId}` 索引
- `queue:events:active` 內的 event 標記

這會造成：

- 前方還有幾個人的計算失真
- 某些 event 明明沒人在排隊，仍被當作 active
- 使用者重新加入 queue 時被舊資料干擾

### queue timeout cleanup 目前會做的事

- 找出 `ExpiredAt` 已到期的 queue snapshot
- 刪除對應的 `purchase_token` 關聯
- 刪除 `queue:user:event:{eventId}:{userId}`
- 從 `queue:event:{eventId}` 移除該 `queueToken`
- 將 snapshot 狀態改成 `expired`
- 如果 event queue 已空，從 `queue:events:active` 移除該 event
