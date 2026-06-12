ALTER TABLE users
ADD COLUMN password_hash VARCHAR(255) NOT NULL DEFAULT '';

UPDATE users
SET password_hash = 'PENDING_RESET'
WHERE password_hash = '';
