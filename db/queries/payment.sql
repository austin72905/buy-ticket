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

-- name: UpdatePaymentStatus :exec
UPDATE payments
SET
    status = $2,
    paid_at = $3,
    failed_at = $4,
    updated_at = $5
WHERE id = $1;
