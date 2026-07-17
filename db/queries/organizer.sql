-- name: GetOrganizerByID :one
SELECT
    id,
    name,
    status,
    created_at,
    updated_at
FROM organizers
WHERE id = $1
LIMIT 1;

-- name: ListOrganizers :many
SELECT
    id,
    name,
    status,
    created_at,
    updated_at
FROM organizers
ORDER BY id;

-- name: CreateOrganizer :one
INSERT INTO organizers (
    name,
    status
) VALUES (
    $1, $2
)
RETURNING
    id,
    name,
    status,
    created_at,
    updated_at;

-- name: UpdateOrganizer :one
UPDATE organizers
SET
    name = $2,
    status = $3,
    updated_at = $4
WHERE id = $1
RETURNING
    id,
    name,
    status,
    created_at,
    updated_at;
