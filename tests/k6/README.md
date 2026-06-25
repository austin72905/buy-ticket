# k6 Booking Flow

這組腳本用來測主要搶票流程，並把註冊、登入、搶票拆開，避免正式壓測時把 auth 成本混進搶票流程。

`booking-flow.js` 會在 `setup()` 階段檢查：

```text
GET /healthz
GET /events/{eventId}/availability
GET /sale/status
```

正式 VU 搶票流程只跑：

```text
POST /queue/join
-> GET /queue/status/{queueToken}
-> POST /reservations
-> POST /orders
```

不包含付款，也不在正式壓測流程中註冊或登入。

## 前置條件

後端請使用 dev 環境，確保會走 Postgres、Redis queue、Redis stock：

```powershell
make run-dev
```

建議壓測前先清乾淨測試交易資料與 Redis：

```powershell
docker exec buy-ticket-postgres psql -U postgres -d buy_ticket -c "TRUNCATE TABLE payment_attempts, idempotency_keys, payments, orders, reservations RESTART IDENTITY CASCADE; UPDATE event_sections SET reserved_quantity = 0, sold_quantity = 0, status = 1, updated_at = now();"
docker exec buy-ticket-redis redis-cli FLUSHDB
```

清完後重啟 `make run-dev`，讓啟動流程用 DB 重建 Redis stock。

## 固定測試用戶

用戶規則放在：

```text
tests/k6/users.json
```

預設準備 1000 個用戶：

```text
k6-user-1@load.local
k6-user-2@load.local
...
k6-user-1000@load.local
```

密碼預設：

```text
password123
```

## 第一步：註冊固定用戶

只需在第一次或清掉 users 後執行：

```powershell
k6 run -e BASE_URL=http://localhost:8080 -e USER_COUNT=1000 -e REGISTER_VUS=50 tests/k6/register-users.js
```

這個腳本可以重複執行；如果使用者已存在，會改用 login 確認帳密可用。

## 第二步：預先登入並產生 sessions

每次正式壓測前執行：

```powershell
go run ./cmd/k6-login-sessions
```

預設會登入 1000 個固定用戶，並產生：

```text
tests/k6/sessions.json
```

如需覆蓋設定：

```powershell
$env:BASE_URL="http://localhost:8080"
$env:USER_COUNT="1000"
$env:LOGIN_CONCURRENCY="50"
go run ./cmd/k6-login-sessions
```

`sessions.json` 是產物檔，已被 `.gitignore` 排除。

## 第三步：執行搶票壓測

```powershell
k6 run -e BASE_URL=http://localhost:8080 -e EVENT_ID=1 -e QUANTITY=1 -e VUS=1000 -e MAX_QUEUE_POLLS=120 tests/k6/booking-flow.js
```

`booking-flow.js` 會直接讀 `tests/k6/sessions.json` 裡的 session cookie，正式壓測階段不會打 `/auth/register` 或 `/auth/login`。

`/queue/join` 有 app 層 backpressure。當同時進入 queue join 的 request 超過設定值時，後端會回：

```http
429 Too Many Requests
Retry-After: 1
```

這比 TCP refused 更可控，前端或壓測腳本可以依 `Retry-After` 重試。相關設定：

```properties
queue.join.max_in_flight=500
queue.join.retry_after_seconds=1
```

這個保護是單一 app instance 內的 in-flight 上限；如果未來開多個 app instance 放在 Load Balancer 後面，整體可承接量會近似變成每個 instance 上限加總，但 Redis/Postgres 仍是共享瓶頸。

壓測腳本對 `/queue/join` 會重試暫時性失敗：

- TCP request error，k6 status 為 `0`
- `429 Too Many Requests`
- `5xx`

預設最多重試 5 次，每次間隔 1 秒；如果後端回 `Retry-After`，會優先使用後端指定秒數。

## Section 選擇

### 固定買單一區

指定 `SECTION_ID` 時，所有 VU 都會買同一區。

```powershell
k6 run -e BASE_URL=http://localhost:8080 -e EVENT_ID=1 -e SECTION_ID=1 -e QUANTITY=1 -e VUS=1000 -e MAX_QUEUE_POLLS=120 tests/k6/booking-flow.js
```

### 指定多個區輪流買

指定 `SECTION_IDS` 時，VU 會在這些區域 round-robin 分散購買。

```powershell
k6 run -e BASE_URL=http://localhost:8080 -e EVENT_ID=1 -e SECTION_IDS=1,2,3 -e QUANTITY=1 -e VUS=1000 -e MAX_QUEUE_POLLS=120 tests/k6/booking-flow.js
```

### 自動分散到所有可買區

不傳 `SECTION_ID` 和 `SECTION_IDS` 時，腳本會從 `/events/{eventId}/availability` 取得所有 `available_quantity >= QUANTITY` 的區域，然後依 VU round-robin 分散購買。

## 參數說明

- `BASE_URL`：後端 API base URL，預設 `http://localhost:8080`。
- `EVENT_ID`：活動 ID，預設 `1`。
- `SECTION_ID`：固定購買單一區域；有設定時優先於 `SECTION_IDS`。
- `SECTION_IDS`：逗號分隔的區域 ID，例如 `1,2,3`。
- `QUANTITY`：每個 VU 購買張數，預設 `1`。
- `VUS`：虛擬使用者數量，預設 `20`。
- `MAX_QUEUE_POLLS`：最多輪詢 queue status 次數，預設 `30`。
- `QUEUE_POLL_SECONDS`：每次輪詢間隔秒數，預設 `1`。
- `QUEUE_JOIN_RETRIES`：`/queue/join` 暫時性失敗最多重試次數，預設 `5`。
- `QUEUE_JOIN_RETRY_SECONDS`：`/queue/join` 重試間隔秒數，預設 `1`。
- `HOLD_MINUTES`：reservation hold_until 時間，預設 `10`。
- `RUN_ID`：測試批次 ID；不設定時用當下 timestamp。
- `MAX_DURATION`：scenario 最大時間，預設 `2m`。
- `USER_COUNT`：固定測試用戶數量，預設讀 `tests/k6/users.json`。
- `USER_OFFSET`：使用者起始偏移，預設 `0`；例如 `USER_OFFSET=1000` 會從 `k6-user-1001` 開始用。
- `REGISTER_VUS`：註冊固定用戶腳本的併發數，預設 `50`。
- `LOGIN_CONCURRENCY`：產生 sessions 時的登入併發數，預設 `50`。
- `SESSION_FILE`：booking flow 讀取的 session 檔，預設 `./sessions.json`。

## 指標與 request tags

主要指標：

- `booking_flow_success_rate`
- `reservation_success_rate`
- `order_success_rate`
- `queue_ready_rate`
- `active_reservation_errors`
- `purchase_token_errors`
- `section_stock_errors`
- `queue_join_retries`
- `http_req_duration` p95 / p99

每個 HTTP request 都有 `name` tag，方便拆開看哪個 API 慢：

- `healthz`
- `availability`
- `sale_status`
- `queue_join`
- `queue_status`
- `reserve`
- `create_order`
- `register_user`
- `existing_user_login`

如果要輸出 JSON summary 後分析，可以加：

```powershell
k6 run -e BASE_URL=http://localhost:8080 -e EVENT_ID=1 -e QUANTITY=1 -e VUS=1000 -e MAX_QUEUE_POLLS=120 --summary-export tests/k6/summary.json tests/k6/booking-flow.js
```

## DB 對帳

跑完後可以查區域庫存：

```sql
SELECT
    id,
    section_name,
    total_quantity,
    reserved_quantity,
    sold_quantity,
    total_quantity - reserved_quantity - sold_quantity AS available_quantity
FROM event_sections
WHERE event_id = 1
ORDER BY id;
```

也可以查 `event_sections` 與 `reservations` 是否一致：

```sql
SELECT
    s.id,
    s.section_name,
    s.reserved_quantity AS section_reserved,
    COALESCE(SUM(r.quantity) FILTER (WHERE r.status = 1), 0) AS holding_reserved,
    s.sold_quantity AS section_sold,
    COALESCE(SUM(r.quantity) FILTER (WHERE r.status = 2), 0) AS confirmed_sold
FROM event_sections s
LEFT JOIN reservations r
    ON r.event_id = s.event_id
   AND r.section_id = s.id
WHERE s.event_id = 1
GROUP BY s.id, s.section_name, s.reserved_quantity, s.sold_quantity
ORDER BY s.id;
```

預期：

- 剛跑完且尚未付款：`section_reserved` 應等於 `holding_reserved`。
- 訂單過期後：未付款訂單會被回收，`section_reserved` 應回到 0。
- 付款成功後：`section_reserved` 會下降，`section_sold` 會上升。
