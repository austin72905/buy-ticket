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

-- name: ListAdminEvents :many
SELECT
    e.id,
    e.organizer_id,
    o.name AS organizer_name,
    o.status AS organizer_status,
    e.name,
    e.venue,
    e.status,
    e.start_at,
    e.end_at,
    e.sale_start_at,
    e.sale_end_at,
    e.created_at,
    e.updated_at
FROM events e
LEFT JOIN organizers o ON o.id = e.organizer_id
WHERE sqlc.arg(organizer_id)::bigint = 0
   OR e.organizer_id = sqlc.arg(organizer_id)
ORDER BY e.id;

-- name: GetAdminEventByID :one
SELECT
    e.id,
    e.organizer_id,
    o.name AS organizer_name,
    o.status AS organizer_status,
    e.name,
    e.venue,
    e.status,
    e.start_at,
    e.end_at,
    e.sale_start_at,
    e.sale_end_at,
    e.created_at,
    e.updated_at
FROM events e
LEFT JOIN organizers o ON o.id = e.organizer_id
WHERE e.id = sqlc.arg(event_id)
  AND (
      sqlc.arg(organizer_id)::bigint = 0
      OR e.organizer_id = sqlc.arg(organizer_id)
  )
LIMIT 1;

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

-- name: AdvanceEventStatuses :execrows
UPDATE events
SET
    status = CASE
        WHEN status = 2 AND sale_end_at < sqlc.arg(now)::timestamptz THEN 4
        WHEN status = 2
             AND sale_start_at <= sqlc.arg(now)::timestamptz
             AND sale_end_at >= sqlc.arg(now)::timestamptz THEN 3
        WHEN status = 3 AND sale_end_at < sqlc.arg(now)::timestamptz THEN 4
        ELSE status
    END,
    updated_at = sqlc.arg(now)::timestamptz
WHERE
    (status = 2 AND sale_end_at < sqlc.arg(now)::timestamptz)
    OR (
        status = 2
        AND sale_start_at <= sqlc.arg(now)::timestamptz
        AND sale_end_at >= sqlc.arg(now)::timestamptz
    )
    OR (status = 3 AND sale_end_at < sqlc.arg(now)::timestamptz);

-- name: UpdateEvent :one
UPDATE events
SET
    organizer_id = $2,
    name = $3,
    venue = $4,
    status = $5,
    start_at = $6,
    end_at = $7,
    sale_start_at = $8,
    sale_end_at = $9,
    updated_at = $10
WHERE id = $1
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
