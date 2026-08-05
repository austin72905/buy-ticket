# buy-ticket 啟動與部署方式

這份文件說明本 repository 的本機啟動、image build，以及目前實際使用的 Helm deployment repository。

## 1. Runtime 架構

應用使用同一個 binary 與 Docker image，透過 `APP_ROLE` 決定啟動內容。

| `APP_ROLE` | 用途 | 行為 |
| --- | --- | --- |
| `all` 或未設定 | 本機開發預設 | API 與 scheduler 都啟動 |
| `api` | Kubernetes API Deployment | 只啟動 HTTP API |
| `scheduler` | Kubernetes scheduler Deployment | 只啟動背景任務，不監聽 HTTP port |

Kubernetes 通常以同一個 image 啟動兩個 Deployment：

- `buy-ticket-api`：`APP_ROLE=api`，可水平擴充。
- `buy-ticket-scheduler`：`APP_ROLE=scheduler`，目前建議維持單一 replica。

Scheduler 只有 process-local no-overlap guard，尚未使用 distributed lock 或 leader election；多個 scheduler Pod 仍可能同時執行相同 job。

## 2. 本機開發

### 啟動 infrastructure

```powershell
cd D:\SourceCode\Go\buy-ticket
make up
```

本 repository 的 `docker-compose.yml` 只啟動：

- PostgreSQL：`localhost:5432`
- Redis：`localhost:6379`
- RabbitMQ：`localhost:5672`，管理介面為 `localhost:15672`

Docker Compose 不會啟動 Go API、scheduler、Vue frontend 或 mock payment service。RabbitMQ 目前也尚未接入應用流程；outbox publisher 現階段只寫入 log。

### 執行 migration

```powershell
make migrate-up
```

### 啟動 API 與 scheduler

```powershell
make run-dev
```

`make run-dev` 會使用 `APP_ENV=dev`。未設定 `APP_ROLE` 時等同 `APP_ROLE=all`。

只啟動 API：

```powershell
$env:APP_ROLE="api"
make run-dev
Remove-Item Env:APP_ROLE
```

只啟動 scheduler：

```powershell
$env:APP_ROLE="scheduler"
make run-dev
Remove-Item Env:APP_ROLE
```

### Mock payment service

Mock payment service 是外部相依服務，不在本 repository，也不會由 `docker compose` 啟動。付款流程預設連線到：

```text
http://localhost:8081
```

啟動付款流程前，需另行啟動相容的 mock payment service，並確認 callback 可以連回：

```text
http://localhost:8080/payments/provider/ecpay/callback
```

## 3. Docker image build

`go-infra` 使用 Go module tag dependency，目前版本為 `github.com/austin72905/go-infra v0.1.1`，Docker build 不需要本機 sibling repository。

```powershell
cd D:\SourceCode\Go\buy-ticket
docker build -t buy-ticket:local .
```

### Migration image

Release 可以另外建立 migration image：

```text
ghcr.io/austin72905/buy-ticket-migrate:<tag>
```

image 內容包含：

```text
/usr/local/bin/migrate
/app/migrations
```

本機建立方式：

```powershell
docker build -f Dockerfile.migrate -t buy-ticket-migrate:local .
```

是否以及如何在部署時執行 migration，應以 `buy-ticket-deploy` repository 的 chart 與 values 設定為準。

## 4. Helm 部署來源

目前實際部署不使用本 repository 的 `charts/buy-ticket`。正式部署設定由獨立 repository 管理：

- [austin72905/buy-ticket-deploy](https://github.com/austin72905/buy-ticket-deploy)

Chart 位於 deployment repository 根目錄，主要設定檔為 `values.yaml`。部署設定、image tag、replica、Secret、migration 與環境差異都應在該 repository 維護。

取得 deployment repository：

```bash
git clone https://github.com/austin72905/buy-ticket-deploy.git
cd buy-ticket-deploy
```

部署前先檢查實際 `values.yaml`，再渲染 chart：

```bash
helm template buy-ticket . -f ./values.yaml
```

安裝或升級：

```bash
helm upgrade --install buy-ticket . \
  --namespace buy-ticket \
  --create-namespace \
  -f ./values.yaml
```

本 repository 內的 `charts/buy-ticket` 僅保留為舊版或參考用 chart，不是目前 deployment source of truth，不應用它修改正式環境。

## 5. 部署設定注意事項

Pod 裡的 `localhost` 指向 Pod 本身，不能用來連線主機上的 PostgreSQL、Redis 或 mock payment service。部署 values 應填寫 Pod 可存取的 Kubernetes Service、DNS 名稱或外部位址。

正式環境至少需要正確設定：

- `POSTGRES_DSN`
- `REDIS_ADDR`
- `PAYMENT_MOCK_BASE_URL`
- `PAYMENT_MOCK_CALLBACK_URL`
- payment signature secrets

其他注意事項：

- API replicas 可以水平擴充。
- Scheduler 目前維持一個 replica，避免相同排程跨 Pod 重複執行。
- 若 scheduler 要高可用，應先加入 leader election、distributed lock 或資料 claim 機制。
- 正式密碼與簽章資料不要直接寫入公開的 `values.yaml`，應透過 Kubernetes Secret 或外部 secret 管理。
- 部署前確認 migration 已包含在 release 流程，避免新程式先於資料庫 schema 上線。

## 6. 部署後檢查

```bash
kubectl get pods -n buy-ticket
kubectl get svc -n buy-ticket
kubectl logs deployment/buy-ticket-api -n buy-ticket
kubectl logs deployment/buy-ticket-scheduler -n buy-ticket
```

實際 resource name 與 namespace 若由 deployment chart 覆寫，以上指令需依 `buy-ticket-deploy/values.yaml` 調整。
