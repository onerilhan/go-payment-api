CREATE TABLE balances (
    user_id INTEGER PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    amount DECIMAL(15,2) NOT NULL DEFAULT 0.00 CHECK (amount >= 0),
    last_updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Performance için index
CREATE INDEX idx_balances_last_updated ON balances(last_updated_at);

-- Trigger: balance güncellendiğinde last_updated_at'i otomatik güncelle
CREATE OR REPLACE FUNCTION update_balance_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_balance_timestamp
    BEFORE UPDATE ON balances
    FOR EACH ROW
    EXECUTE FUNCTION update_balance_timestamp();