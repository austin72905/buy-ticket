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

-- name: GetOrderByIDForUpdate :one
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
FOR UPDATE;

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

-- name: UpdateOrderStatusIfCurrent :execrows
UPDATE orders
SET
    status = sqlc.arg(status),
    paid_at = sqlc.narg(paid_at),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND status = sqlc.arg(expected_status);

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

-- name: ListAdminOrders :many
SELECT
    o.id,
    o.order_no,
    o.reservation_id,
    o.reservation_no,
    o.event_id,
    o.event_name,
    o.section_id,
    o.section_name,
    o.user_id,
    o.user_name,
    u.email AS user_email,
    o.quantity,
    o.unit_price,
    o.total_amount,
    o.status,
    o.expires_at,
    o.paid_at,
    o.created_at,
    o.updated_at
FROM orders o
JOIN events e ON e.id = o.event_id
JOIN users u ON u.id = o.user_id
WHERE (sqlc.arg(organizer_id)::bigint = 0 OR e.organizer_id = sqlc.arg(organizer_id))
  AND (sqlc.arg(event_id)::bigint = 0 OR o.event_id = sqlc.arg(event_id))
  AND (sqlc.arg(user_id)::bigint = 0 OR o.user_id = sqlc.arg(user_id))
  AND (sqlc.arg(status)::smallint = 0 OR o.status = sqlc.arg(status))
  AND (
      sqlc.arg(cursor_created_at)::timestamptz IS NULL
      OR o.created_at < sqlc.arg(cursor_created_at)
      OR (o.created_at = sqlc.arg(cursor_created_at) AND o.id < sqlc.arg(cursor_id))
  )
ORDER BY o.created_at DESC, o.id DESC
LIMIT sqlc.arg(page_limit);

-- name: GetAdminOrderSensitiveByID :one
SELECT
    o.id,
    o.order_no,
    o.reservation_id,
    o.reservation_no,
    o.event_id,
    o.event_name,
    o.section_id,
    o.section_name,
    o.user_id,
    o.user_name,
    u.email AS user_email,
    o.quantity,
    o.unit_price,
    o.total_amount,
    o.status,
    o.expires_at,
    o.paid_at,
    o.created_at,
    o.updated_at
FROM orders o
JOIN users u ON u.id = o.user_id
WHERE o.id = $1
LIMIT 1;
