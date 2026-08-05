# Typed Config 設定說明

## 設計方向

目前應用程式改為使用 `config.go` 的 typed config 載入設定，不再依賴 embedded `app.properties` 與 `runtime.Property.Validate()`。

啟動時會執行：

1. 讀取 `.env`：只在 `APP_ENV` 未設定、`local`、`dev` 時讀取。
2. 讀取環境變數：環境變數優先於 `.env`。
3. 套用程式內預設值。
4. 執行必要設定驗證。

Production / Kubernetes 建議直接用 Deployment env、ConfigMap、Secret 注入，不需要 `.env`。

## 本機啟動

第一次本機啟動可以先建立 `.env`：

```powershell
Copy-Item .env.example .env
```

然後啟動：

```powershell
make run-dev
```

`make run-dev` 會設定 `APP_ENV=dev`，現在 `dev` 也會讀取 `.env`。如果沒有 `.env`，程式會使用 dev 預設值：

- `QUEUE_STORE=redis`
- `REDIS_ADDR=localhost:6379`
- `ORDER_PAYMENT_TTL_MINUTES=3`
- mock payment 簽章設定使用測試值

## Runtime Role

`APP_ROLE` 控制同一個 binary 的啟動模式：

| APP_ROLE | 行為 |
| --- | --- |
| `all` 或未設定 | API + scheduler，適合本機開發 |
| `api` | 只啟動 HTTP API |
| `scheduler` | 只啟動背景排程，不 listen HTTP port |

Kubernetes 建議同一個 image 部署兩個 Deployment：

- API Deployment：`APP_ROLE=api`
- Scheduler Deployment：`APP_ROLE=scheduler`

Scheduler job 目前有本地 no-overlap guard：

- 同一個 scheduler Pod 內，同一個 job 如果上一輪還沒跑完，下一輪會直接跳過。
- 這可以避免每秒或短週期 job 在單一 process 內重入。
- 這不是 Redis distributed lock；如果 scheduler replicas 大於 1，不同 Pod 之間仍可能同時執行同一個 job。
- 目前 Kubernetes 建議維持 `scheduler.replicaCount=1`。若未來要 scheduler HA，再補 Redis lock 或 leader election。

## 主要環境變數

| 變數 | 說明 |
| --- | --- |
| `SERVER_ADDR` | HTTP listen address，例如 `:8080` |
| `PPROF_ENABLED` | 是否開啟 `/debug/pprof` |
| `QUEUE_STORE` | `memory` 或 `redis` |
| `QUEUE_RELEASE_LIMIT` | 每次 scheduler 放行到 ready 的人數上限 |
| `QUEUE_JOIN_MAX_IN_FLIGHT` | 單一 API process 內 `/queue/join` 最大同時處理數 |
| `ORDER_PAYMENT_TTL_MINUTES` | 訂單待付款時間 |
| `SESSION_TTL_HOURS` | 前台 / 後台 session TTL |
| `POSTGRES_DSN` | PostgreSQL 連線字串 |
| `REDIS_ADDR` | Redis address；`QUEUE_STORE=redis` 時必填 |
| `PAYMENT_MOCK_BASE_URL` | mock payment primary provider URL |
| `PAYMENT_MOCK_BACKUP_BASE_URL` | mock payment backup provider URL |
| `PAYMENT_MOCK_CALLBACK_URL` | mock payment callback 回打 buy-ticket 的 URL |
| `PAYMENT_BREAKER_*` | payment provider circuit breaker 設定 |
| `PAYMENT_RECONCILE_*` | payment attempt 補償查詢排程設定 |
| `OUTBOX_PUBLISH_*` | outbox 發送排程設定 |

完整範例請看 `.env.example`。

## 驗證規則

typed config 只驗證必要條件：

- `SERVER_ADDR` 不可空
- `POSTGRES_DSN` 不可空
- `QUEUE_STORE` 只能是 `memory` 或 `redis`
- `QUEUE_STORE=redis` 時，`REDIS_ADDR` 不可空

這樣避免舊版 properties validator 因為不同 `APP_ROLE` 沒用到某些設定而啟動失敗。
