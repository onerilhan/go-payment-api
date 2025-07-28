-- Remove indexes
DROP INDEX IF EXISTS idx_balance_history_user_date;
DROP INDEX IF EXISTS idx_balance_history_transaction_id;
DROP INDEX IF EXISTS idx_balance_history_created_at;
DROP INDEX IF EXISTS idx_balance_history_user_id;

-- Remove table
DROP TABLE IF EXISTS balance_history;