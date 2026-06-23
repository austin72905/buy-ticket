# k6 Booking Flow

這支腳本測主要搶票流程：

```text
register/login
-> GET /sale/status
-> POST /queue/join
-> GET /queue/status/{queueToken}
-> POST /reservations
-> POST /orders
```

腳本不壓付款，目標是先驗證排隊、purchase token、reservation、order 建立與庫存不超賣。

## 執行前

建議用 `dev` 環境，讓後端使用 Postgres、Redis queue、Redis stock：

```powershell
make run-dev
```

確認 event 正在販售，且指定 section 有足夠庫存。

## 執行

```powershell
k6 run tests/k6/booking-flow.js
```

常用參數：

```powershell
k6 run `
  -e BASE_URL=http://localhost:8080 `
  -e EVENT_ID=1 `
  -e SECTION_ID=1 `
  -e QUANTITY=1 `
  -e VUS=50 `
  -e MAX_QUEUE_POLLS=60 `
  tests/k6/booking-flow.js
```

如果不傳 `SECTION_ID`，腳本會從 `/events/{eventId}/availability` 選第一個可買區域。

## 注意

- 每個 VU 只跑一次，避免同一個 user 對同一個 event 重複建立 active reservation。
- 每次執行會用新的 `RUN_ID` 產生測試帳號與訂單編號。
- 腳本會建立 pending payment orders；如果不付款，庫存會先被 reservation/order hold 住，等 order expire job 回收。
- 如果要重跑大量測試，建議先準備足夠票數，或清理測試資料。

## 建議觀察

- `booking_flow_success_rate`
- `reservation_success_rate`
- `order_success_rate`
- `queue_ready_rate`
- `active_reservation_errors`
- `purchase_token_errors`
- `section_stock_errors`
- `http_req_duration` p95 / p99
- 測後查 DB 是否 oversell：

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
