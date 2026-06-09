INSERT INTO users (
    id,
    name,
    email
) VALUES
    (1, 'Austin Lin', 'austin@example.com');

INSERT INTO events (
    id,
    name,
    venue,
    status,
    start_at,
    end_at,
    sale_start_at,
    sale_end_at
) VALUES (
    1,
    'Sample Concert',
    'Taipei Arena',
    3,
    NOW() + INTERVAL '30 days',
    NOW() + INTERVAL '30 days 2 hours',
    NOW() - INTERVAL '1 hour',
    NOW() + INTERVAL '1 day'
);

INSERT INTO event_sections (
    id,
    event_id,
    event_name,
    section_name,
    price,
    total_quantity,
    reserved_quantity,
    sold_quantity,
    purchase_limit,
    status
) VALUES
    (
        1,
        1,
        'Sample Concert',
        'A 區',
        2800,
        100,
        0,
        0,
        4,
        1
    ),
    (
        2,
        1,
        'Sample Concert',
        'B 區',
        1800,
        150,
        0,
        0,
        4,
        1
    );

SELECT setval(pg_get_serial_sequence('users', 'id'), COALESCE((SELECT MAX(id) FROM users), 1), true);
SELECT setval(pg_get_serial_sequence('events', 'id'), COALESCE((SELECT MAX(id) FROM events), 1), true);
SELECT setval(pg_get_serial_sequence('event_sections', 'id'), COALESCE((SELECT MAX(id) FROM event_sections), 1), true);
