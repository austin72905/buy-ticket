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

-- name: ListAdminEventSections :many
SELECT
    s.id,
    s.event_id,
    s.event_name,
    s.section_name,
    s.price,
    s.total_quantity,
    s.reserved_quantity,
    s.sold_quantity,
    s.purchase_limit,
    s.status,
    s.created_at,
    s.updated_at
FROM event_sections s
JOIN events e ON e.id = s.event_id
WHERE s.event_id = sqlc.arg(event_id)
  AND (
      sqlc.arg(organizer_id)::bigint = 0
      OR e.organizer_id = sqlc.arg(organizer_id)
  )
ORDER BY s.id;

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

-- name: UpdateSection :one
UPDATE event_sections
SET
    section_name = $3,
    price = $4,
    total_quantity = $5,
    purchase_limit = $6,
    status = $7,
    updated_at = $8
WHERE event_id = $1
  AND id = $2
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
