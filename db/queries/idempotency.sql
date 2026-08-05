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
WHERE user_id = $1
  AND key = $2
  AND endpoint = $3
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
    status = sqlc.arg(status),
    response_status = sqlc.narg(response_status),
    response_body = sqlc.narg(response_body),
    locked_until = NULL,
    updated_at = sqlc.arg(updated_at)
WHERE user_id = sqlc.arg(user_id)
  AND key = sqlc.arg(key)
  AND endpoint = sqlc.arg(endpoint);
