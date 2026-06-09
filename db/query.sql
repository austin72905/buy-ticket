-- name: GetEventByID :one
SELECT
    id,
    name,
    venue,
    status,
    start_at,
    end_at,
    sale_start_at,
    sale_end_at,
    created_at,
    updated_at
FROM events
WHERE id = $1
LIMIT 1;

-- name: ListEvents :many
SELECT
    id,
    name,
    venue,
    status,
    start_at,
    end_at,
    sale_start_at,
    sale_end_at,
    created_at,
    updated_at
FROM events
ORDER BY id;

-- name: CreateEvent :one
INSERT INTO events (
    name,
    venue,
    status,
    start_at,
    end_at,
    sale_start_at,
    sale_end_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING
    id,
    name,
    venue,
    status,
    start_at,
    end_at,
    sale_start_at,
    sale_end_at,
    created_at,
    updated_at;

-- name: GetSectionByEventAndID :one
SELECT
    id,
    event_id,
    event_name,
    section_name,
    price,
    total_quantity,
    reserved_quantity,
    sold_quantity,
    purchase_limit,
    status,
    created_at,
    updated_at
FROM event_sections
WHERE event_id = $1
  AND id = $2
LIMIT 1;

-- name: ListSectionsByEventID :many
SELECT
    id,
    event_id,
    event_name,
    section_name,
    price,
    total_quantity,
    reserved_quantity,
    sold_quantity,
    purchase_limit,
    status,
    created_at,
    updated_at
FROM event_sections
WHERE event_id = $1
ORDER BY id;

-- name: CreateSection :one
INSERT INTO event_sections (
    event_id,
    event_name,
    section_name,
    price,
    total_quantity,
    reserved_quantity,
    sold_quantity,
    purchase_limit,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING
    id,
    event_id,
    event_name,
    section_name,
    price,
    total_quantity,
    reserved_quantity,
    sold_quantity,
    purchase_limit,
    status,
    created_at,
    updated_at;

-- name: UpdateSectionInventory :exec
UPDATE event_sections
SET
    reserved_quantity = $2,
    sold_quantity = $3,
    status = $4,
    updated_at = $5
WHERE id = $1;

-- name: GetReservationByID :one
SELECT
    id,
    reservation_no,
    event_id,
    event_name,
    section_id,
    section_name,
    user_id,
    user_name,
    quantity,
    unit_price,
    total_amount,
    status,
    expires_at,
    created_at,
    updated_at
FROM reservations
WHERE id = $1
LIMIT 1;

-- name: GetReservationByReservationNo :one
SELECT
    id,
    reservation_no,
    event_id,
    event_name,
    section_id,
    section_name,
    user_id,
    user_name,
    quantity,
    unit_price,
    total_amount,
    status,
    expires_at,
    created_at,
    updated_at
FROM reservations
WHERE reservation_no = $1
LIMIT 1;

-- name: CreateReservation :one
INSERT INTO reservations (
    reservation_no,
    event_id,
    event_name,
    section_id,
    section_name,
    user_id,
    user_name,
    quantity,
    unit_price,
    total_amount,
    status,
    expires_at,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)
RETURNING
    id,
    reservation_no,
    event_id,
    event_name,
    section_id,
    section_name,
    user_id,
    user_name,
    quantity,
    unit_price,
    total_amount,
    status,
    expires_at,
    created_at,
    updated_at;

-- name: UpdateReservationStatus :exec
UPDATE reservations
SET
    status = $2,
    updated_at = $3
WHERE id = $1;

-- name: ListExpiredHoldingReservations :many
SELECT
    id,
    reservation_no,
    event_id,
    event_name,
    section_id,
    section_name,
    user_id,
    user_name,
    quantity,
    unit_price,
    total_amount,
    status,
    expires_at,
    created_at,
    updated_at
FROM reservations
WHERE status = 1
  AND expires_at <= $1
ORDER BY expires_at
LIMIT $2;

-- name: ListReservationsByUserID :many
SELECT
    id,
    reservation_no,
    event_id,
    event_name,
    section_id,
    section_name,
    user_id,
    user_name,
    quantity,
    unit_price,
    total_amount,
    status,
    expires_at,
    created_at,
    updated_at
FROM reservations
WHERE user_id = $1
ORDER BY created_at DESC, id DESC;

-- name: GetOrderByID :one
SELECT
    id,
    order_no,
    reservation_id,
    reservation_no,
    event_id,
    event_name,
    section_id,
    section_name,
    user_id,
    user_name,
    quantity,
    unit_price,
    total_amount,
    status,
    expires_at,
    paid_at,
    created_at,
    updated_at
FROM orders
WHERE id = $1
LIMIT 1;

-- name: GetOrderByOrderNo :one
SELECT
    id,
    order_no,
    reservation_id,
    reservation_no,
    event_id,
    event_name,
    section_id,
    section_name,
    user_id,
    user_name,
    quantity,
    unit_price,
    total_amount,
    status,
    expires_at,
    paid_at,
    created_at,
    updated_at
FROM orders
WHERE order_no = $1
LIMIT 1;

-- name: CreateOrder :one
INSERT INTO orders (
    order_no,
    reservation_id,
    reservation_no,
    event_id,
    event_name,
    section_id,
    section_name,
    user_id,
    user_name,
    quantity,
    unit_price,
    total_amount,
    status,
    expires_at,
    paid_at,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
)
RETURNING
    id,
    order_no,
    reservation_id,
    reservation_no,
    event_id,
    event_name,
    section_id,
    section_name,
    user_id,
    user_name,
    quantity,
    unit_price,
    total_amount,
    status,
    expires_at,
    paid_at,
    created_at,
    updated_at;

-- name: UpdateOrderStatus :exec
UPDATE orders
SET
    status = $2,
    paid_at = $3,
    updated_at = $4
WHERE id = $1;

-- name: ListExpiredPendingOrders :many
SELECT
    id,
    order_no,
    reservation_id,
    reservation_no,
    event_id,
    event_name,
    section_id,
    section_name,
    user_id,
    user_name,
    quantity,
    unit_price,
    total_amount,
    status,
    expires_at,
    paid_at,
    created_at,
    updated_at
FROM orders
WHERE status = 1
  AND expires_at <= $1
ORDER BY expires_at
LIMIT $2;

-- name: ListOrdersByUserID :many
SELECT
    id,
    order_no,
    reservation_id,
    reservation_no,
    event_id,
    event_name,
    section_id,
    section_name,
    user_id,
    user_name,
    quantity,
    unit_price,
    total_amount,
    status,
    expires_at,
    paid_at,
    created_at,
    updated_at
FROM orders
WHERE user_id = $1
ORDER BY created_at DESC, id DESC;

-- name: CreatePayment :one
INSERT INTO payments (
    payment_no,
    order_id,
    order_no,
    reservation_id,
    event_id,
    event_name,
    user_id,
    user_name,
    method,
    amount,
    status,
    paid_at,
    failed_at,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
)
RETURNING
    id,
    payment_no,
    order_id,
    order_no,
    reservation_id,
    event_id,
    event_name,
    user_id,
    user_name,
    method,
    amount,
    status,
    paid_at,
    failed_at,
    created_at,
    updated_at;

-- name: GetPaymentByPaymentNo :one
SELECT
    id,
    payment_no,
    order_id,
    order_no,
    reservation_id,
    event_id,
    event_name,
    user_id,
    user_name,
    method,
    amount,
    status,
    paid_at,
    failed_at,
    created_at,
    updated_at
FROM payments
WHERE payment_no = $1
LIMIT 1;

-- name: UpdatePaymentStatus :exec
UPDATE payments
SET
    status = $2,
    paid_at = $3,
    failed_at = $4,
    updated_at = $5
WHERE id = $1;
