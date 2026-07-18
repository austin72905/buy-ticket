ALTER TABLE payment_attempts
ADD COLUMN reconcile_attempts INTEGER NOT NULL DEFAULT 0,
ADD COLUMN next_reconcile_at TIMESTAMPTZ NULL,
ADD COLUMN last_reconcile_error TEXT NULL;

CREATE INDEX idx_payment_attempts_reconcile_due
ON payment_attempts (next_reconcile_at, created_at, id)
WHERE status IN (1, 4);
