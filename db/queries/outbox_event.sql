-- name: CreateOutboxEvent :one
INSERT INTO outbox_events (
    event_id,
    event_type,
    aggregate_type,
    aggregate_id,
    payload,
    status,
    attempts,
    max_attempts,
    next_attempt_at,
    last_error,
    published_at,
    created_at,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $12
)
RETURNING
    id,
    event_id,
    event_type,
    aggregate_type,
    aggregate_id,
    payload,
    status,
    attempts,
    max_attempts,
    next_attempt_at,
    last_error,
    published_at,
    created_at,
    updated_at;

-- name: ListPendingOutboxEvents :many
SELECT
    id,
    event_id,
    event_type,
    aggregate_type,
    aggregate_id,
    payload,
    status,
    attempts,
    max_attempts,
    next_attempt_at,
    last_error,
    published_at,
    created_at,
    updated_at
FROM outbox_events
WHERE status = 1
  AND attempts < max_attempts
  AND (
      next_attempt_at IS NULL
      OR next_attempt_at <= sqlc.arg(now)::timestamptz
  )
ORDER BY created_at ASC, id ASC
LIMIT sqlc.arg(limit_rows)::integer;

-- name: UpdateOutboxEventPublishState :exec
UPDATE outbox_events
SET
    status = $2,
    attempts = $3,
    next_attempt_at = $4,
    last_error = $5,
    published_at = $6,
    updated_at = $7
WHERE id = $1;
