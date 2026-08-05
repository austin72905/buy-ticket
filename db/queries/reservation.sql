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

-- name: GetReservationByIDForUpdate :one
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
FOR UPDATE;

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

-- name: UpdateReservationStatusIfCurrent :execrows
UPDATE reservations
SET
    status = sqlc.arg(status),
    updated_at = sqlc.arg(updated_at)
WHERE id = sqlc.arg(id)
  AND status = sqlc.arg(expected_status);

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
