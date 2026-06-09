DELETE FROM event_sections
WHERE id IN (1, 2);

DELETE FROM events
WHERE id = 1;

DELETE FROM users
WHERE id = 1;
