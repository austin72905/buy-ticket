-- name: GetUserByID :one
SELECT
    id,
    name,
    email,
    password_hash,
    created_at,
    updated_at
FROM users
WHERE id = $1
LIMIT 1;

-- name: GetUserByEmail :one
SELECT
    id,
    name,
    email,
    password_hash,
    created_at,
    updated_at
FROM users
WHERE email = $1
LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (
    name,
    email,
    password_hash
) VALUES (
    $1, $2, $3
)
RETURNING
    id,
    name,
    email,
    password_hash,
    created_at,
    updated_at;

-- name: UpdateUserPassword :exec
UPDATE users
SET
    password_hash = $2,
    updated_at = $3
WHERE id = $1;

-- name: GetAdminUserByID :one
SELECT
    id,
    organizer_id,
    name,
    email,
    password_hash,
    role,
    status,
    created_at,
    updated_at
FROM admin_users
WHERE id = $1
LIMIT 1;

-- name: GetAdminUserByEmail :one
SELECT
    id,
    organizer_id,
    name,
    email,
    password_hash,
    role,
    status,
    created_at,
    updated_at
FROM admin_users
WHERE email = $1
LIMIT 1;

-- name: CreateAdminUser :one
INSERT INTO admin_users (
    organizer_id,
    name,
    email,
    password_hash,
    role,
    status
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING
    id,
    organizer_id,
    name,
    email,
    password_hash,
    role,
    status,
    created_at,
    updated_at;

-- name: CreateAdminAuditLog :one
INSERT INTO admin_audit_logs (
    admin_user_id,
    action,
    target_type,
    target_id,
    reason,
    ip_address,
    user_agent
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING
    id,
    admin_user_id,
    action,
    target_type,
    target_id,
    reason,
    ip_address,
    user_agent,
    created_at;

-- name: GetEventByID :one
SELECT
    id,
    organizer_id,
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
    organizer_id,
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
    organizer_id,
    name,
    venue,
    status,
    start_at,
    end_at,
    sale_start_at,
    sale_end_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING
    id,
    organizer_id,
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

-- name: ReserveSectionInventory :one
UPDATE event_sections
SET
    reserved_quantity = reserved_quantity + sqlc.arg(quantity)::integer,
    updated_at = sqlc.arg(updated_at)
WHERE event_id = sqlc.arg(event_id)
  AND id = sqlc.arg(id)
  AND status = 1
  AND sqlc.arg(quantity)::integer > 0
  AND (purchase_limit = 0 OR sqlc.arg(quantity)::integer <= purchase_limit)
  AND reserved_quantity + sold_quantity + sqlc.arg(quantity)::integer <= total_quantity
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

-- name: ReleaseSectionInventory :one
UPDATE event_sections
SET
    reserved_quantity = reserved_quantity - sqlc.arg(quantity)::integer,
    status = CASE
        WHEN status = 3 AND total_quantity - (reserved_quantity - sqlc.arg(quantity)::integer) - sold_quantity > 0 THEN 1
        ELSE status
    END,
    updated_at = sqlc.arg(updated_at)
WHERE event_id = sqlc.arg(event_id)
  AND id = sqlc.arg(id)
  AND sqlc.arg(quantity)::integer > 0
  AND reserved_quantity >= sqlc.arg(quantity)::integer
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

-- name: ConfirmSectionSale :one
UPDATE event_sections
SET
    reserved_quantity = reserved_quantity - sqlc.arg(quantity)::integer,
    sold_quantity = sold_quantity + sqlc.arg(quantity)::integer,
    status = CASE
        WHEN total_quantity - (reserved_quantity - sqlc.arg(quantity)::integer) - (sold_quantity + sqlc.arg(quantity)::integer) = 0 THEN 3
        ELSE status
    END,
    updated_at = sqlc.arg(updated_at)
WHERE event_id = sqlc.arg(event_id)
  AND id = sqlc.arg(id)
  AND sqlc.arg(quantity)::integer > 0
  AND reserved_quantity >= sqlc.arg(quantity)::integer
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

-- name: GetActiveReservationByUserAndEvent :one
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
  AND event_id = $2
  AND status = 1
  AND expires_at > $3
ORDER BY created_at DESC, id DESC
LIMIT 1;

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

-- name: ListPaymentsByUserID :many
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
WHERE user_id = $1
ORDER BY created_at DESC, id DESC;

-- name: CreatePaymentAttempt :one
INSERT INTO payment_attempts (
    order_id,
    payment_id,
    idempotency_key,
    provider,
    merchant_trade_no,
    provider_trade_no,
    method,
    amount,
    status,
    request_payload,
    response_payload,
    callback_payload,
    failure_reason,
    expires_at,
    succeeded_at,
    failed_at,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $17
)
RETURNING
    id,
    order_id,
    payment_id,
    idempotency_key,
    provider,
    merchant_trade_no,
    provider_trade_no,
    method,
    amount,
    status,
    request_payload,
    response_payload,
    callback_payload,
    failure_reason,
    expires_at,
    succeeded_at,
    failed_at,
    created_at,
    updated_at;

-- name: GetPaymentAttemptByMerchantTradeNo :one
SELECT
    id,
    order_id,
    payment_id,
    idempotency_key,
    provider,
    merchant_trade_no,
    provider_trade_no,
    method,
    amount,
    status,
    request_payload,
    response_payload,
    callback_payload,
    failure_reason,
    expires_at,
    succeeded_at,
    failed_at,
    created_at,
    updated_at
FROM payment_attempts
WHERE merchant_trade_no = $1
LIMIT 1;

-- name: GetPaymentAttemptByIdempotencyKey :one
SELECT
    id,
    order_id,
    payment_id,
    idempotency_key,
    provider,
    merchant_trade_no,
    provider_trade_no,
    method,
    amount,
    status,
    request_payload,
    response_payload,
    callback_payload,
    failure_reason,
    expires_at,
    succeeded_at,
    failed_at,
    created_at,
    updated_at
FROM payment_attempts
WHERE idempotency_key = $1
LIMIT 1;

-- name: ListPaymentAttemptsByOrderID :many
SELECT
    id,
    order_id,
    payment_id,
    idempotency_key,
    provider,
    merchant_trade_no,
    provider_trade_no,
    method,
    amount,
    status,
    request_payload,
    response_payload,
    callback_payload,
    failure_reason,
    expires_at,
    succeeded_at,
    failed_at,
    created_at,
    updated_at
FROM payment_attempts
WHERE order_id = $1
ORDER BY created_at DESC, id DESC;

-- name: UpdatePaymentAttemptStatus :exec
UPDATE payment_attempts
SET
    payment_id = $2,
    provider_trade_no = $3,
    status = $4,
    response_payload = $5,
    callback_payload = $6,
    failure_reason = $7,
    succeeded_at = $8,
    failed_at = $9,
    updated_at = $10
WHERE id = $1;

-- name: GetIdempotencyKey :one
SELECT
    id,
    key,
    user_id,
    endpoint,
    request_hash,
    status,
    response_status,
    response_body,
    locked_until,
    expires_at,
    created_at,
    updated_at
FROM idempotency_keys
WHERE key = $1
  AND endpoint = $2
LIMIT 1;

-- name: CreateIdempotencyKey :one
INSERT INTO idempotency_keys (
    key,
    user_id,
    endpoint,
    request_hash,
    status,
    locked_until,
    expires_at,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $8
)
RETURNING
    id,
    key,
    user_id,
    endpoint,
    request_hash,
    status,
    response_status,
    response_body,
    locked_until,
    expires_at,
    created_at,
    updated_at;

-- name: CompleteIdempotencyKey :exec
UPDATE idempotency_keys
SET
    status = $3,
    response_status = $4,
    response_body = $5,
    locked_until = NULL,
    updated_at = $6
WHERE key = $1
  AND endpoint = $2;

-- name: UpdatePaymentStatus :exec
UPDATE payments
SET
    status = $2,
    paid_at = $3,
    failed_at = $4,
    updated_at = $5
WHERE id = $1;
