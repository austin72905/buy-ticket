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

-- name: ListAdminUsers :many
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
ORDER BY id;

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

-- name: UpdateAdminUser :one
UPDATE admin_users
SET
    organizer_id = $2,
    name = $3,
    email = $4,
    password_hash = $5,
    role = $6,
    status = $7,
    updated_at = $8
WHERE id = $1
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
