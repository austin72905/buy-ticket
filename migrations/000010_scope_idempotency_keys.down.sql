DROP INDEX idx_payment_attempts_order_id_idempotency_key;

CREATE UNIQUE INDEX idx_payment_attempts_idempotency_key
ON payment_attempts (idempotency_key)
WHERE idempotency_key IS NOT NULL;

DROP INDEX idx_idempotency_keys_user_key_endpoint;

CREATE UNIQUE INDEX idx_idempotency_keys_key_endpoint
ON idempotency_keys (key, endpoint);

ALTER TABLE idempotency_keys
ALTER COLUMN user_id DROP NOT NULL;
