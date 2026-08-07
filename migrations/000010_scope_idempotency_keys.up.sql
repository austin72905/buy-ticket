DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM idempotency_keys WHERE user_id IS NULL) THEN
        RAISE EXCEPTION 'idempotency_keys contains rows with NULL user_id; backfill or remove them before applying migration 000010';
    END IF;
END
$$;

ALTER TABLE idempotency_keys
ALTER COLUMN user_id SET NOT NULL;

DROP INDEX idx_idempotency_keys_key_endpoint;

CREATE UNIQUE INDEX idx_idempotency_keys_user_key_endpoint
ON idempotency_keys (user_id, key, endpoint);

DROP INDEX idx_payment_attempts_idempotency_key;

CREATE UNIQUE INDEX idx_payment_attempts_order_id_idempotency_key
ON payment_attempts (order_id, idempotency_key)
WHERE idempotency_key IS NOT NULL;
