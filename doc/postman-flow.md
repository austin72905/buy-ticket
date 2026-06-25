# Postman Flow

## 1. Setup

### 1.1 Start Dependencies

```bash
make up
make migrate-up
```

### 1.2 Start API

Memory mode:

```bash
make run-local
```

PostgreSQL mode:

1. Edit `config/local/app.properties`.
2. Set `app.store=postgres`.
3. Configure `postgres.dsn`.
4. Restart the API.

```bash
make run-local
```

### 1.3 Base URL

```text
http://localhost:8080
```

---

## 2. Main Booking Flow

### Step 1: Register or Login

Register:

```http
POST /auth/register
Content-Type: application/json
```

```json
{
  "name": "Postman User",
  "email": "postman@example.com",
  "password": "password123"
}
```

Or login:

```http
POST /auth/login
Content-Type: application/json
```

```json
{
  "email": "postman@example.com",
  "password": "password123"
}
```

The response sets the `buy_ticket_session` cookie. Keep cookies enabled in Postman for the following authenticated requests.

### Step 2: List Events

```http
GET /events
```

Check the response and keep the `event_id`.

### Step 3: Check Sections and Availability

```http
GET /events/1/sections
GET /events/1/availability
```

Check the response and keep the `section_id` and available quantity.

### Step 4: Join Queue and Get Purchase Token

```http
POST /queue/join
Content-Type: application/json
```

```json
{
  "event_id": 1,
  "client_id": "postman-device-001",
  "request_id": "req-001",
  "channel": "web"
}
```

Expected fields:

- `queue_token`
- `purchase_token`, when the queue status is ready

If the queue is still waiting, poll:

```http
GET /queue/status/{queueToken}
```

### Step 5: Create Reservation

```http
POST /reservations
Content-Type: application/json
```

```json
{
  "event_id": 1,
  "section_id": 1,
  "quantity": 2,
  "hold_until": "2026-06-10T19:00:00+08:00",
  "purchase_token": "pt_xxx"
}
```

Expected fields:

- `id`
- `status = holding`
- `total_amount`

### Step 6: Check Reservations

```http
GET /reservations/{reservationId}
GET /me/reservations
```

### Step 7: Create Order

```http
POST /orders
Content-Type: application/json
```

```json
{
  "reservation_id": 1,
  "order_no": "ORD-TEST-001",
  "purchase_token": "pt_xxx"
}
```

Expected fields:

- `id`
- `status = pending_payment`

### Step 8: Check Orders

```http
GET /orders/{orderId}
GET /orders/order-no/ORD-TEST-001
GET /me/orders
```

### Step 9: Pay Order

Demo direct-pay endpoint:

```http
POST /payments
Idempotency-Key: pay-test-001
Content-Type: application/json
```

```json
{
  "order_id": 1,
  "method": "credit_card"
}
```

Checks:

- `Idempotency-Key` is currently optional for demo compatibility, but the frontend should send it before real payment provider integration.
- The frontend only sends `order_id` and `method`.
- The backend loads the order and sets `amount = order.total_amount`.
- The backend generates `payment_no`.
- The backend sets `paid_at` from server time.
- The backend marks the payment paid, marks the order paid, and confirms the reservation.

Provider-flow start endpoint:

```http
POST /payments/start
Idempotency-Key: pay-start-test-001
Content-Type: application/json
```

```json
{
  "order_id": 1,
  "method": "credit_card",
  "provider": "mock_ecpay"
}
```

Checks:

- Returns `202 Accepted`.
- Creates a `payment_attempt`.
- Calls mock pay service at `payment.mock.base_url`, default `http://localhost:8081`.
- If mock pay service is not running, the attempt is marked `timeout`.
- Does not mark the order paid yet.
- Does not confirm the reservation yet.
- Mock pay service callback will complete the payment.

### Step 10: Check Payment

```http
GET /payments/PAY-TEST-001
GET /me/payments
```

Expected final state:

- `payment.status = paid`
- `order.status = paid`
- `reservation.status = confirmed`

---

## 3. Common Query APIs

```http
GET /healthz
GET /sale/status?event_id=1
GET /events
GET /events/1
GET /events/1/sections
GET /events/1/availability
GET /queue/status/{queueToken}
GET /reservations/{reservationId}
GET /orders/{orderId}
GET /orders/order-no/{orderNo}
GET /payments/{paymentNo}
GET /me
GET /me/reservations
GET /me/orders
GET /me/payments
```

---

## 4. Common Error Cases

### 4.1 Missing Purchase Token

`POST /reservations` response:

```json
{
  "error": "purchase token is required"
}
```

### 4.2 Purchase Token Not Found

```json
{
  "error": "purchase token not found"
}
```

### 4.3 Duplicate Queue Join

```json
{
  "error": "user already joined queue"
}
```

### 4.4 Missing Login

Authenticated APIs return:

```json
{
  "error": "unauthorized"
}
```
