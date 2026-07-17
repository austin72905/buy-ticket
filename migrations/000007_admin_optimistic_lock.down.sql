ALTER TABLE event_sections
DROP COLUMN version;

ALTER TABLE events
DROP COLUMN version;

ALTER TABLE admin_users
DROP COLUMN version;

ALTER TABLE organizers
DROP COLUMN version;
