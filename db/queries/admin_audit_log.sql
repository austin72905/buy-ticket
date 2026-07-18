-- name: CreateAdminAuditLog :one
INSERT INTO admin_audit_logs (
    admin_user_id,
    action,
    target_type,
    target_id,
    reason,
    ip_address,
    user_agent
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING
    id,
    admin_user_id,
    action,
    target_type,
    target_id,
    reason,
    ip_address,
    user_agent,
    created_at;

-- name: ListAdminAuditLogs :many
SELECT
    l.id,
    l.admin_user_id,
    au.name AS admin_user_name,
    au.email AS admin_user_email,
    au.role AS admin_user_role,
    au.status AS admin_user_status,
    l.action,
    l.target_type,
    l.target_id,
    CASE
        WHEN l.target_type = 'ADMIN_USER' THEN target_admin.name
        WHEN l.target_type = 'ORGANIZER' THEN target_organizer.name
        WHEN l.target_type = 'EVENT' THEN target_event.name
        WHEN l.target_type = 'SECTION' THEN target_section.section_name
        WHEN l.target_type = 'ORDER' THEN target_order.order_no
        ELSE NULL
    END AS target_name,
    l.reason,
    l.ip_address,
    l.user_agent,
    l.created_at
FROM admin_audit_logs l
LEFT JOIN admin_users au ON au.id = l.admin_user_id
LEFT JOIN admin_users target_admin ON l.target_type = 'ADMIN_USER' AND target_admin.id = l.target_id
LEFT JOIN organizers target_organizer ON l.target_type = 'ORGANIZER' AND target_organizer.id = l.target_id
LEFT JOIN events target_event ON l.target_type = 'EVENT' AND target_event.id = l.target_id
LEFT JOIN event_sections target_section ON l.target_type = 'SECTION' AND target_section.id = l.target_id
LEFT JOIN orders target_order ON l.target_type = 'ORDER' AND target_order.id = l.target_id
WHERE (sqlc.arg(admin_user_id)::bigint = 0 OR l.admin_user_id = sqlc.arg(admin_user_id))
  AND (
      sqlc.arg(cursor_created_at)::timestamptz IS NULL
      OR l.created_at < sqlc.arg(cursor_created_at)
      OR (l.created_at = sqlc.arg(cursor_created_at) AND l.id < sqlc.arg(cursor_id))
  )
ORDER BY l.created_at DESC, l.id DESC
LIMIT sqlc.arg(page_limit);
