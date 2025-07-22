DROP TRIGGER IF EXISTS update_balance_timestamp ON balances;
DROP FUNCTION IF EXISTS update_balance_timestamp();
DROP INDEX IF EXISTS idx_balances_last_updated;
DROP TABLE IF EXISTS balances;