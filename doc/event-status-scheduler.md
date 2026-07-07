# Event Status Scheduler

## 目的

活動是否可以排隊買票，不只看開賣時間，也會看活動狀態。

目前後端判斷可售條件：

```text
event.status == ON_SALE
AND now >= sale_start_at
AND now <= sale_end_at
```

所以如果活動時間到了，但狀態仍是 `DRAFT` 或 `PUBLISHED`，仍然會回：

```json
{
  "code": "EVENT_NOT_ON_SALE",
  "message": "event is not on sale"
}
```

## 自動推進規則

`APP_ROLE=all` 或 `APP_ROLE=scheduler` 啟動時會跑 event status scheduler。

規則：

| 目前狀態 | 條件 | 新狀態 |
| --- | --- | --- |
| `DRAFT` | 任意時間 | 不自動變更 |
| `PUBLISHED` | `sale_start_at <= now <= sale_end_at` | `ON_SALE` |
| `PUBLISHED` | `sale_end_at < now` | `ENDED` |
| `ON_SALE` | `sale_end_at < now` | `ENDED` |
| `ENDED` | 任意時間 | 不自動變更 |

## 為什麼 DRAFT 不自動開賣

`DRAFT` 代表活動尚未準備好，可能票區、價格、文案還沒確認。

只有後台已上架成 `PUBLISHED` 的活動，才會在時間到時自動轉成 `ON_SALE`。

## 本機驗證

確認 scheduler 有啟動：

```powershell
make run-dev
```

或只啟動 scheduler：

```powershell
$env:APP_ROLE="scheduler"
make run-dev
Remove-Item Env:APP_ROLE
```

手動檢查活動狀態：

```sql
SELECT id, name, status, sale_start_at, sale_end_at
FROM events
ORDER BY id;
```

狀態數值：

| 數值 | API 字串 | 意義 |
| --- | --- | --- |
| `1` | `DRAFT` | 草稿 |
| `2` | `PUBLISHED` | 已上架，等待自動開賣 |
| `3` | `ON_SALE` | 開售中 |
| `4` | `ENDED` | 已結束 |

