DROP INDEX IF EXISTS idx_payment_attempts_reconcile_due;

ALTER TABLE payment_attempts
DROP COLUMN last_reconcile_error,
DROP COLUMN next_reconcile_at,
DROP COLUMN reconcile_attempts;
