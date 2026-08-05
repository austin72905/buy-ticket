# Queue Design

## 目標

排隊系統的目標是把瞬間大量搶票請求切成可控流量，避免所有使用者同時進入 reservation / stock 扣庫存流程。

目前設計重點：

- `/queue/join` 只負責加入 waiting queue。
- scheduler 依照 `QUEUE_RELEASE_LIMIT` 逐批放行。
- 被放行的使用者會取得 `purchase_token`。
- `/reservations` 必須帶 `purchase_token` 才能建立 reservation。
- `purchase_token` 成功消費後會失效，避免同一個放行資格重複使用。

## 主流程

```text
user
  -> POST /queue/join
  -> GET /queue/status/{queueToken}
  -> status=ready + purchase_token
  -> POST /reservations
  -> POST /orders
  -> POST /payments/start
```

前端不應該在排隊時直接打 reservation。reservation 是被 queue 放行後的下一步。

## Queue Status

目前 queue status 使用字串表示：

| Status | 意義 |
| --- | --- |
| `WAITING` | 已加入 waiting queue，尚未放行 |
| `READY` | 已放行，可以使用 `purchase_token` 建立 reservation |
| `EXPIRED` | queue token 或 purchase token 已過期 |

`READY` 不代表已訂到票，只代表取得進入 reservation 階段的資格。

## API Contract

### POST /queue/join

用途：加入指定活動的排隊隊列。

Request：

```json
{
  "event_id": 1
}
```

Response：

```json
{
  "queue_token": "qt_20260612233742.971083100",
  "status": "WAITING",
  "ahead_count": 10
}
```

如果活動已經允許立即放行，也可能直接回：

```json
{
  "queue_token": "qt_20260612233742.971083100",
  "status": "READY",
  "ahead_count": 0,
  "purchase_token": "pt_20260613012918.860164300",
  "purchase_token_expires_at": "2026-06-13T01:34:18+08:00"
}
```

### GET /queue/status/{queueToken}

用途：查詢目前排隊狀態。

WAITING：

```json
{
  "queue_token": "qt_20260612233742.971083100",
  "status": "WAITING",
  "ahead_count": 5
}
```

READY：

```json
{
  "queue_token": "qt_20260612233742.971083100",
  "status": "READY",
  "ahead_count": 0,
  "purchase_token": "pt_20260613012918.860164300",
  "purchase_token_expires_at": "2026-06-13T01:34:18+08:00"
}
```

## Redis Model

目前 Redis queue 使用 waiting / ready 分離模型。

```text
queue:waiting:{eventID}
queue:ready:{eventID}
queue:snapshot:{queueToken}
queue:purchase:{purchaseToken}
queue:user:{eventID}:{userID}
queue:events:active
queue:seq:{eventID}
```

### waiting queue

```text
queue:waiting:{eventID}
```

- Redis ZSET。
- score 使用遞增 sequence，確保先加入的人先被放行。
- `/queue/join` 會寫入 waiting queue。

### ready queue

```text
queue:ready:{eventID}
```

- Redis ZSET。
- scheduler promote 後，token 會從 waiting 移到 ready。
- ready queue 用來限制同時持有 `purchase_token` 的人數。

### queue snapshot

```text
queue:snapshot:{queueToken}
```

保存 queue token 對應狀態，例如：

- event id
- user id
- status
- sequence
- purchase token
- expires at

### purchase token mapping

```text
queue:purchase:{purchaseToken}
```

用途是讓 `/reservations` 可以從 `purchase_token` 找回 queue snapshot，並驗證：

- token 存在
- token 尚未過期
- token 屬於目前登入 user
- token 屬於指定 event
- token 尚未被消費

## Scheduler Promote

queue 放行由 scheduler 控制，不由 `GET /queue/status` 即時計算放行。

概念流程：

```text
readyCount = ZCARD queue:ready:{eventID}
slots = QUEUE_RELEASE_LIMIT - readyCount
tokens = ZPOPMIN queue:waiting:{eventID} slots

for each token:
  load snapshot
  status = READY
  generate purchase_token
  ZADD queue:ready:{eventID}
  SET queue:purchase:{purchaseToken}
```

這樣可以避免每次使用者 poll queue status 都掃整條 queue。

目前 scheduler job 有 local no-overlap guard：同一個 Pod 內，如果上一輪 job 還沒跑完，下一輪會 skip。

## Purchase Token

`purchase_token` 是「被放行後的 reservation 資格」，不是訂單，也不是付款憑證。

特性：

- 有有效期限。
- 成功建立 reservation 後會被 consume。
- consume 後不能重複使用。
- cancel reservation 不應該取消整個 queue token；重選座位應該在 reservation 層處理。

需要 `purchase_token` 的原因：

- queue token 只代表排隊身份。
- purchase token 代表已被放行。
- reservation API 可以明確驗證使用者是否真的被放行。

## Cleanup

### purchase token cleanup

用途：清掉已放行但逾時未建立 reservation 的 token。

會處理：

- 過期 `purchase_token`
- ready queue 中過期 token
- snapshot 內過期狀態

### queue timeout cleanup

用途：清掉長時間未被放行或使用者離開的 queue token。

會處理：

- waiting queue 過期 token
- ready queue 過期 token
- user / event 去重 key
- active event index

不能只靠 Redis TTL，因為 token 過期後還需要同步清理 ZSET 與關聯 key。

## 錯誤案例

活動不可售：

```json
{
  "code": "EVENT_NOT_ON_SALE",
  "message": "event is not on sale",
  "request_id": "..."
}
```

未登入：

```json
{
  "code": "UNAUTHORIZED",
  "message": "unauthorized",
  "request_id": "..."
}
```

purchase token 不存在：

```json
{
  "code": "BAD_REQUEST",
  "message": "purchase token not found",
  "request_id": "..."
}
```

## 限制與後續補強

目前設計仍有幾個可補強點：

- scheduler Pod 之間尚未使用 Redis distributed lock，目前建議 `scheduler.replicaCount=1`。
- promote 流程若要更嚴謹，可以用 Lua script 收斂成單次 Redis 原子操作。
- queue join backpressure 目前是 process-local，若 API 多 Pod，可再補 Redis token bucket。

這些限制不影響目前單 scheduler Pod 的主流程，但要做 scheduler HA 時需要補上。
