# API Flow

This document describes the current `buy-ticket` booking flow:

- auth session
- queue
- purchase token
- reservation
- order
- payment
- background jobs
- stock reconciliation

---

## 1. Main Flow

1. Register or login first.
2. Check sale status with `GET /sale/status`.
3. Join queue with `POST /queue/join`.
4. Poll queue status with `GET /queue/status/{queueToken}`.
5. Background job promotes waiting users to ready.
6. Ready queue status returns a `purchase_token`.
7. Create reservation with `POST /reservations` and the `purchase_token`.
8. Create order with `POST /orders`.
9. Start provider payment with `POST /payments/start`; frontend should send `Idempotency-Key`.
10. Background jobs clean up expired queues, purchase tokens, unpaid orders, and Redis stock drift.

Authenticated write APIs read the current user from the session cookie. The frontend must not send `user_id` for queue join or reservation creation.

---

## 2. Flow Diagram

```mermaid
flowchart TD
    A["POST /auth/login"] --> B["GET /sale/status?event_id=1"]
    B --> C{"Can join queue?"}
    C -- "No" --> X["Show unavailable state"]
    C -- "Yes" --> D["POST /queue/join"]
    D --> E["Return queue_token"]
    E --> F["Poll GET /queue/status/{queueToken}"]
    W["Queue Background Job"] --> G{"Can promote?"}
    G -- "No" --> F
    G -- "Yes" --> H["queue status = ready"]
    H --> I["Return purchase_token"]
    I --> J["POST /reservations"]
    J --> K{"purchase_token valid?"}
    K -- "No" --> Y["Return error"]
    K -- "Yes" --> L["Create reservation"]
    L --> M["POST /orders"]
    M --> N["Create pending_payment order"]
    N --> O["POST /payments/start"]
    O --> P["Create payment_attempt"]
    P --> Q["Call mock payment service"]
    Q --> T{"Provider callback successful?"}
    T -- "No" --> U["Wait for retry or expiration job"]
    T -- "Yes" --> R["order = paid"]
    R --> S["reservation = confirmed"]
```

---

## 3. Sequence Diagram

```mermaid
sequenceDiagram
    participant Client as Client
    participant API as buy-ticket API
    participant Queue as Queue Store
    participant Job as Background Job
    participant Booking as Booking Service

    Client->>API: POST /auth/login
    API-->>Client: Set-Cookie buy_ticket_session

    Client->>API: GET /sale/status?event_id=1
    API-->>Client: is_on_sale / can_join_queue

    Client->>API: POST /queue/join
    API->>Queue: Join(...)
    Queue-->>API: queue_token / waiting or ready
    API-->>Client: join response

    loop polling
        Client->>API: GET /queue/status/{queueToken}
        API->>Queue: Get(...)
        Queue-->>API: waiting / ready
        API-->>Client: queue status
    end

    Job->>Queue: PromoteReady(...)
    Queue-->>Job: promote waiting users to ready

    Client->>API: POST /reservations + purchase_token
    API->>Booking: ReserveTicket(...)
    Booking->>Queue: ConsumePurchaseToken(...)
    Booking-->>API: reservation
    API-->>Client: reservation response

    Client->>API: POST /orders
    API->>Booking: CreateOrder(...)
    Booking-->>API: order
    API-->>Client: order response

    Client->>API: POST /payments/start + Idempotency-Key
    API->>Booking: Create payment_attempt
    Booking->>Booking: amount = order.total_amount
    Booking->>Booking: Generate merchant_trade_no
    Booking->>Payment: POST /api/payment/process
    Payment-->>Booking: processing response
    Booking-->>API: payment_attempt
    API-->>Client: 202 Accepted

    Payment->>API: POST /payments/provider/ecpay/callback
    API->>Booking: Find payment_attempt by MerchantTradeNo
    API->>Booking: PayOrder(...)
    Booking-->>API: payment / order paid / reservation confirmed
    API-->>Payment: callback accepted

    Job->>Booking: SweepExpiredOrders(...)
    Booking-->>Job: expired orders count

    Job->>Booking: ReconcileStock(...)
    Booking-->>Job: checked / fixed section count
```

---

## 4. Queue Background Job

Queue promotion is controlled by a background job, not by `GET /queue/status`.

Behavior:

- Runs every 1 second.
- Calls `QueueStore.PromoteReady(...)`.
- Uses `queue.release.limit` to control how many users can enter `ready` each run.

Configuration:

- Job name: `queue-promote-ready`
- Cron spec: `*/1 * * * * *`
- `queue.release.limit`: loaded from properties

---

## 5. Order Expiration Job

Unpaid order expiration is controlled by a background job.

Behavior:

- Runs every 5 seconds.
- Calls `BookingService.SweepExpiredOrders(...)`.
- Finds expired `pending_payment` orders.
- Marks the order as `expired`.
- Marks the reservation as `expired`.
- Releases section stock.

Configuration:

- Job name: `order-expire-sweep`
- Cron spec: `*/5 * * * * *`
- `order.expire.batch.size`: loaded from properties
- `order.payment.ttl_minutes`: server-side pending payment TTL used when creating orders

---

## 6. Stock Reconciliation Job

Redis stock reconciliation is controlled by a background job.

Behavior:

- Runs every 1 minute.
- Calls `BookingService.ReconcileStock(...)`.
- Loads section availability from PostgreSQL.
- Compares Redis stock with DB calculated availability.
- Fixes Redis stock when a mismatch is found.

Configuration:

- Job name: `stock-reconcile`
- Cron spec: `0 * * * * *`
- Details: see `doc/stock-reconciliation.md`

---

## 7. Queue Status

Queue status uses an integer enum:

- `1` = waiting
- `2` = ready
- `3` = expired

---

## 8. Key APIs

Public APIs:

- `POST /auth/register`
- `POST /auth/login`
- `GET /sale/status`
- `GET /queue/status/{queueToken}`
- `GET /events`
- `GET /events/{eventId}`
- `GET /events/{eventId}/sections`
- `GET /events/{eventId}/availability`
- `POST /payments/provider/ecpay/callback`

Authenticated APIs:

- `POST /auth/logout`
- `GET /me`
- `GET /me/reservations`
- `GET /me/orders`
- `GET /me/payments`
- `POST /queue/join`
- `POST /reservations`
- `POST /orders`
- `POST /payments`
- `POST /payments/start`
- `POST /reservations/expire`
- `POST /reservations/cancel`

---

## 9. Remaining Work

- Queue timeout cleanup
- Purchase token cleanup
- Stock consistency hardening: see `doc/stock-reconciliation.md`
- Payment idempotency: see `doc/payment-idempotency.md`
- Payment attempts and provider start flow: see `doc/payment-attempts.md`
- Payment callback / webhook hardening
- RabbitMQ delay or DLQ timeout flow

---

## 10. Payment API Modes

The project currently keeps two payment entry points:

- `POST /payments`: demo direct-pay flow. It directly creates a paid payment, marks the order paid, and confirms the reservation.
- `POST /payments/start`: provider-style flow. It creates a `payment_attempt` and returns `202 Accepted`; it does not mark the order paid.

This keeps the current demo flow working while adding the provider payment foundation.
