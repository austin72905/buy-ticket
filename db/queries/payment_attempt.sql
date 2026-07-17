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
