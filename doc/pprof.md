# pprof 壓測分析

`pprof` 用來在壓測或問題發生時分析 Go process 內部瓶頸，例如 CPU、heap、goroutine、mutex 與 block。

## 啟用方式

預設關閉：

```properties
pprof.enabled=false
```

本機臨時啟用：

```powershell
$env:PPROF_ENABLED="true"
make run-dev
```

K3s local values 已預設啟用：

```yaml
app:
  pprofEnabled: true
```

正式環境不要把 `/debug/pprof/*` 公開到外網。若要在正式環境使用，建議只透過內網、VPN、admin network 或 `kubectl port-forward` 存取。

## 本機壓測時抓 CPU profile

先啟動 API，再跑 k6。壓測進行中另外開一個 terminal：

```powershell
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
```

常用指令：

```text
top
top -cum
web
list FunctionName
```

## K3s 壓測時抓 CPU profile

先 port-forward 到其中一個 API Pod：

```powershell
kubectl port-forward deploy/buy-ticket-api 8080:8080
```

壓測進行中抓 30 秒 CPU profile：

```powershell
go tool pprof http://localhost:8080/debug/pprof/profile?seconds=30
```

## 常用端點

```text
/debug/pprof/
/debug/pprof/profile?seconds=30
/debug/pprof/heap
/debug/pprof/goroutine
/debug/pprof/mutex
/debug/pprof/block
/debug/pprof/trace?seconds=5
```

## 和 k6 的搭配方式

`k6` 負責產生流量並告訴你哪個 API 慢；`pprof` 負責定位 Go 程式裡哪段 function 消耗最多 CPU 或卡住。

建議流程：

```text
1. 啟動 API 並啟用 pprof
2. 執行 k6 booking flow
3. 壓測中抓 30 秒 CPU profile
4. 用 top / top -cum / web 找瓶頸 function
5. 優化後再次壓測比較 p95 / p99
```

如果 API latency 高但 CPU 不高，下一步應優先看 Redis、Postgres、網路或外部支付服務，而不是只看 CPU profile。
