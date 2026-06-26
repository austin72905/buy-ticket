# buy-ticket 啟動與部署方式

這份文件給新電腦或其他 AI agent 使用，用來快速理解目前專案如何本機啟動、Docker build，以及用 Helm 部署到單機 k3s。

## 1. 目前架構

目前是同一個 repo、同一個 binary、同一個 Docker image，透過 `APP_ROLE` 決定啟動內容。

| APP_ROLE | 用途 | 行為 |
| --- | --- | --- |
| `all` 或未設定 | 本機開發預設 | API + scheduler 都啟動 |
| `api` | Kubernetes API Deployment | 只啟動 HTTP API |
| `scheduler` | Kubernetes scheduler Deployment | 只啟動背景任務，不啟動 HTTP |

Kubernetes 部署時會用同一個 image 啟動兩個 Deployment：

- `buy-ticket-api`：`APP_ROLE=api`，可水平擴充。
- `buy-ticket-scheduler`：`APP_ROLE=scheduler`，replicas 預設 `1`。

## 2. 本機開發啟動

### 啟動 infra

```powershell
cd D:\SourceCode\Go\buy-ticket
make up
```

目前 `docker-compose.yml` 會啟動：

- PostgreSQL：`localhost:5432`
- Redis：`localhost:6379`
- RabbitMQ：`localhost:5672`，管理介面 `localhost:15672`

### 跑 migration

```powershell
make migrate-up
```

### 啟動 API + scheduler

```powershell
make run-dev
```

未設定 `APP_ROLE` 時等同 `APP_ROLE=all`，所以本機開發不需要特別設定。

### 分開測 API / scheduler

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

## 3. Docker image build

`go-infra` 已改成 tag dependency：`github.com/austin72905/go-infra v0.1.0`，所以 Docker build 不需要本機 sibling repo。

```powershell
cd D:\SourceCode\Go\buy-ticket
docker build -t buy-ticket:local .
```

## 4. 單機 k3s 部署

### 前提

目標型態：

```text
同一台 Linux 主機
├─ Docker Compose
│  ├─ PostgreSQL
│  ├─ Redis
│  └─ mock payment service
└─ k3s
   ├─ buy-ticket-api
   └─ buy-ticket-scheduler
```

注意：Pod 裡的 `localhost` 是 Pod 自己，不是 Linux 主機。  
如果 infra 用 Docker Compose 跑在同一台 Linux 主機，`charts/buy-ticket/values-local.yaml` 需要改成 Linux 主機 IP。

查 Linux 主機 IP：

```bash
hostname -I
```

範例：

```yaml
postgres:
  dsn: "postgres://postgres:postgres@192.168.1.10:5432/buy_ticket?sslmode=disable"

redis:
  addr: "192.168.1.10:6379"

payment:
  mock:
    baseURL: "http://192.168.1.10:8081"
    callbackURL: "http://192.168.1.10:8080/payments/provider/ecpay/callback"
```

### 匯入 image 到 k3s

如果沒有 registry，可以先用本機 image 匯入 k3s：

```bash
docker build -t buy-ticket:local .
docker save buy-ticket:local -o buy-ticket-local.tar
sudo k3s ctr images import buy-ticket-local.tar
```

### Helm render 檢查

```bash
helm template buy-ticket ./charts/buy-ticket -f ./charts/buy-ticket/values-local.yaml
```

預期會產生：

- 一個 Service，只 selector `app.kubernetes.io/component: api`
- 一個 `buy-ticket-api` Deployment，`APP_ROLE=api`
- 一個 `buy-ticket-scheduler` Deployment，`APP_ROLE=scheduler`

### Helm 部署

```bash
helm upgrade --install buy-ticket ./charts/buy-ticket -f ./charts/buy-ticket/values-local.yaml
```

### 檢查 Pod

```bash
kubectl get pods
kubectl get svc
kubectl logs deploy/buy-ticket-api
kubectl logs deploy/buy-ticket-scheduler
```

## 5. 常見注意事項

- 本機 `make run-dev` 不需要改，預設仍是 API + scheduler。
- k3s 裡不要把 Postgres / Redis 設成 `localhost`，要填 Pod 能連到的主機 IP 或 Kubernetes service name。
- API replicas 可以調高；scheduler replicas 第一階段維持 `1`，避免重複跑排程。
- 如果之後 scheduler 要高可用，先做 leader election 或 distributed lock，再把 scheduler replicas 調高。
- `values.yaml` / `values-local.yaml` 不要放正式密碼；正式環境用外部 secret 或部署時覆寫。
