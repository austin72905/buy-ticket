# Redis Queue Model

本文件說明目前排隊系統在 Redis 的資料結構。

## 目標

排隊流程採用 waiting queue 與 ready queue 分離的模型，避免 scheduler 每秒掃完整條 queue。

```text
queue:waiting:{eventID}
  尚未放行的 queue token

queue:ready:{eventID}
  已放行、持有 purchase token 的 queue token
```

## Join Queue

使用者呼叫 `POST /queue/join` 時：

1. 建立 `queue_token`
2. 寫入 `queue:waiting:{eventID}`
3. 儲存 `queue:token:{queueToken}` snapshot
4. 回傳 `status=waiting`

Join API 不直接產生 `purchase_token`。

## Scheduler Promote

Scheduler 每秒執行 `queue-promote-ready`。

放行流程：

1. 用 `ZCARD queue:ready:{eventID}` 計算目前 ready 人數
2. 用 `queue.release.limit - readyCount` 算出剩餘可放行名額
3. 從 `queue:waiting:{eventID}` 取出最前面的 token
4. 將 snapshot 改成 `status=ready`
5. 產生 `purchase_token`
6. 寫入 `queue:ready:{eventID}`

這段流程使用 Redis Lua script 收斂 `ZCARD`、`ZPOPMIN`、snapshot 更新、purchase token 建立、ready queue 寫入，降低多 scheduler 或中途失敗造成的不一致風險。

## Queue Status

`GET /queue/status/{queueToken}`：

- 如果 token 在 `queue:waiting:{eventID}`，用 `ZRANK` 計算前方人數。
- 如果 token 在 `queue:ready:{eventID}`，回傳 ready 狀態，`ahead_count=0`。

## Consume Purchase Token

建立訂單時會 consume `purchase_token`：

1. 驗證 `queue:purchase:{purchaseToken}`
2. 清除 purchase token key
3. 從 `queue:ready:{eventID}` 移除 queue token
4. 儲存 snapshot used state

## Cleanup

清理任務分兩種：

- `purchase-token-cleanup`：只掃 `queue:ready:{eventID}`，處理已放行但逾時未建立訂單的人。
- `queue-timeout-cleanup`：掃 `queue:waiting:{eventID}` 與 `queue:ready:{eventID}`，處理整體 queue timeout。

當 waiting queue 與 ready queue 都空了，會從 `queue:events:active` 移除 eventID。

## Key Summary

| Key | 用途 |
| --- | --- |
| `queue:waiting:{eventID}` | 等待放行的 queue token |
| `queue:ready:{eventID}` | 已放行的 queue token |
| `queue:token:{queueToken}` | queue snapshot |
| `queue:purchase:{purchaseToken}` | purchase token 對應 queue token |
| `queue:user:event:{eventID}:{userID}` | 防止同一使用者重複排同一活動 |
| `queue:events:active` | 有活躍 queue 的 eventID |
