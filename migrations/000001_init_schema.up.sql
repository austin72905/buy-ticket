CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_users_email ON users (email);

CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    venue VARCHAR(200) NOT NULL,
    status SMALLINT NOT NULL,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    sale_start_at TIMESTAMPTZ NOT NULL,
    sale_end_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_status_sale_time ON events (status, sale_start_at, sale_end_at);

CREATE TABLE event_sections (
    id BIGSERIAL PRIMARY KEY,
    event_id BIGINT NOT NULL,
    event_name VARCHAR(200) NOT NULL,
    section_name VARCHAR(100) NOT NULL,
    price BIGINT NOT NULL,
    total_quantity INTEGER NOT NULL,
    reserved_quantity INTEGER NOT NULL DEFAULT 0,
    sold_quantity INTEGER NOT NULL DEFAULT 0,
    purchase_limit INTEGER NOT NULL,
    status SMALLINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (price >= 0),
    CHECK (total_quantity >= 0),
    CHECK (reserved_quantity >= 0),
    CHECK (sold_quantity >= 0),
    CHECK (purchase_limit >= 0),
    CHECK (reserved_quantity + sold_quantity <= total_quantity)
);

CREATE INDEX idx_event_sections_event_id_status ON event_sections (event_id, status);
CREATE INDEX idx_event_sections_event_id_section_name ON event_sections (event_id, section_name);

CREATE TABLE reservations (
    id BIGSERIAL PRIMARY KEY,
    reservation_no VARCHAR(40) NOT NULL,
    event_id BIGINT NOT NULL,
    event_name VARCHAR(200) NOT NULL,
    section_id BIGINT NOT NULL,
    section_name VARCHAR(100) NOT NULL,
    user_id BIGINT NOT NULL,
    user_name VARCHAR(100) NOT NULL,
    quantity INTEGER NOT NULL,
    unit_price BIGINT NOT NULL,
    total_amount BIGINT NOT NULL,
    status SMALLINT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (quantity > 0),
    CHECK (unit_price >= 0),
    CHECK (total_amount >= 0)
);

CREATE UNIQUE INDEX idx_reservations_reservation_no ON reservations (reservation_no);
CREATE INDEX idx_reservations_user_id_status_created_at ON reservations (user_id, status, created_at DESC);
CREATE INDEX idx_reservations_event_section_status_expires_at ON reservations (event_id, section_id, status, expires_at);
CREATE INDEX idx_reservations_expires_at_holding ON reservations (expires_at) WHERE status = 1;

CREATE TABLE orders (
    id BIGSERIAL PRIMARY KEY,
    order_no VARCHAR(40) NOT NULL,
    reservation_id BIGINT NOT NULL,
    reservation_no VARCHAR(40) NOT NULL,
    event_id BIGINT NOT NULL,
    event_name VARCHAR(200) NOT NULL,
    section_id BIGINT NOT NULL,
    section_name VARCHAR(100) NOT NULL,
    user_id BIGINT NOT NULL,
    user_name VARCHAR(100) NOT NULL,
    quantity INTEGER NOT NULL,
    unit_price BIGINT NOT NULL,
    total_amount BIGINT NOT NULL,
    status SMALLINT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    paid_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (quantity > 0),
    CHECK (unit_price >= 0),
    CHECK (total_amount >= 0)
);

CREATE UNIQUE INDEX idx_orders_order_no ON orders (order_no);
CREATE UNIQUE INDEX idx_orders_reservation_id ON orders (reservation_id);
CREATE INDEX idx_orders_user_id_status_created_at ON orders (user_id, status, created_at DESC);
CREATE INDEX idx_orders_event_id_status_created_at ON orders (event_id, status, created_at DESC);
CREATE INDEX idx_orders_expires_at_pending ON orders (expires_at) WHERE status = 1;

CREATE TABLE payments (
    id BIGSERIAL PRIMARY KEY,
    payment_no VARCHAR(40) NOT NULL,
    order_id BIGINT NOT NULL,
    order_no VARCHAR(40) NOT NULL,
    reservation_id BIGINT NOT NULL,
    event_id BIGINT NOT NULL,
    event_name VARCHAR(200) NOT NULL,
    user_id BIGINT NOT NULL,
    user_name VARCHAR(100) NOT NULL,
    method VARCHAR(50) NOT NULL,
    amount BIGINT NOT NULL,
    status SMALLINT NOT NULL,
    paid_at TIMESTAMPTZ NULL,
    failed_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (amount >= 0)
);

CREATE UNIQUE INDEX idx_payments_payment_no ON payments (payment_no);
CREATE INDEX idx_payments_order_id_status_created_at ON payments (order_id, status, created_at DESC);
CREATE INDEX idx_payments_user_id_status_created_at ON payments (user_id, status, created_at DESC);
