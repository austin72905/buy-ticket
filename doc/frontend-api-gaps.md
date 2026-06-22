# Frontend API Gaps

The current frontend now connects all backend APIs that already exist:

- `GET /events`
- `GET /events/:eventId`
- `GET /events/:eventId/sections`
- `GET /events/:eventId/availability`
- `GET /sale/status`
- `POST /queue/join`
- `GET /queue/status/:queueToken`
- `POST /reservations`
- `POST /orders`
- `POST /payments/start`

The simplified ticketing UI can run with these endpoints, but several pieces are still rendered with local placeholder data because the backend does not expose them yet.

## 1. Event Display Data

Needed for event list and event detail pages.

Suggested endpoint:

```http
GET /events
GET /events/:eventId
```

Suggested extra fields:

```json
{
  "poster_url": "https://cdn.example.com/posters/event-1.jpg",
  "banner_url": "https://cdn.example.com/banners/event-1.jpg",
  "subtitle": "2026 Taipei Concert",
  "description": "Event introduction text",
  "notice": "Important ticketing notice"
}
```

Current workaround:

- Frontend generates poster-like cards from event name, date, and venue.
- Detail page uses a CSS banner instead of a real event banner.

## 2. Event Notice And Tabs

Needed for the "detailed information before purchase" page.

Suggested endpoint:

```http
GET /events/:eventId/content
```

Suggested response:

```json
{
  "intro": "...",
  "ticket_notice": ["..."],
  "payment_notice": ["..."],
  "pickup_notice": ["..."],
  "refund_notice": ["..."]
}
```

Current workaround:

- Frontend shows static demo notice text.

## 3. Ticket Products Or Sessions

The reference UI shows multiple purchasable rows under one event. The backend currently models sections under an event, but not separate sessions/products.

Suggested endpoint:

```http
GET /events/:eventId/products
```

Suggested response:

```json
[
  {
    "product_id": 1,
    "event_id": 1,
    "name": "General sale",
    "session_time": "2026-08-08T15:10:00+08:00",
    "venue": "Taipei Arena",
    "price_min": 800,
    "price_max": 4680,
    "status": 1
  }
]
```

Current workaround:

- Frontend uses sections as purchasable rows.

## 4. Seat Map

The reference UI has a venue map with colored selectable areas. The backend currently exposes sections and availability, but not map coordinates.

Suggested endpoint:

```http
GET /events/:eventId/seat-map
```

Suggested response:

```json
{
  "venue": "Taipei Arena",
  "stage_label": "Stage",
  "areas": [
    {
      "section_id": 1,
      "name": "A Zone",
      "color": "#2d9bf0",
      "x": 120,
      "y": 80,
      "w": 80,
      "h": 40,
      "shape": "rect"
    }
  ]
}
```

Current workaround:

- Frontend builds a simple CSS seat map from `/availability`.

## 5. Backend Checkout Shortcut

The current frontend calls:

1. `POST /reservations`
2. `POST /orders`
3. `POST /payments/start`

This is good for learning and debugging, but a real UI usually wants a clearer checkout boundary.

Optional endpoint:

```http
POST /checkout/hold-order
```

Suggested request:

```json
{
  "event_id": 1,
  "section_id": 1,
  "quantity": 2,
  "purchase_token": "pt_xxx"
}
```

Suggested response:

```json
{
  "reservation": {},
  "order": {},
  "payment_expires_at": "2026-06-12T20:10:00+08:00"
}
```

Current frontend behavior:

- `Seat / Quantity` page calls `POST /reservations`
- `Cart` page calls `POST /orders`
- `Checkout` page calls `POST /payments/start`
- The calls are intentionally split into separate screens to match the ticketing flow.

## 6. Payment Provider Start

The frontend now uses `POST /payments/start` to create a `payment_attempt`.
The backend calls the mock payment service, then waits for provider callback to mark the order paid.

Provider-flow endpoint:

```http
POST /payments/start
```

Suggested request:

```json
{
  "order_id": 1,
  "method": "credit_card",
  "provider": "mock_ecpay"
}
```

Suggested response:

```json
{
  "id": 1,
  "order_id": 1,
  "provider": "mock_ecpay",
  "merchant_trade_no": "MT-1-20260622120000",
  "method": "credit_card",
  "amount": 2800,
  "status": 1
}
```

Current frontend behavior:

- Frontend starts the mock provider payment with `POST /payments/start`.
- Frontend sends `Idempotency-Key` so retry does not create duplicate attempts.
- Frontend polls the order and payment list briefly after starting the attempt.
- Manual mock provider flow is documented in `doc/mock-payment-callback.md`.
