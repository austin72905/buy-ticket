ALTER TABLE events
DROP CONSTRAINT IF EXISTS fk_events_organizer_id;

DROP INDEX IF EXISTS idx_events_organizer_id_status;

ALTER TABLE events
DROP COLUMN IF EXISTS organizer_id;

DROP TABLE IF EXISTS admin_audit_logs;
DROP TABLE IF EXISTS admin_users;
DROP TABLE IF EXISTS organizers;
