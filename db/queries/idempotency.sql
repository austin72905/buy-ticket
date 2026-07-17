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
