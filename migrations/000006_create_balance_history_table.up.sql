-- Create balance_history table for tracking balance changes
CREATE TABLE balance_history (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    previous_amount DECIMAL(15,2) NOT NULL,
    new_amount DECIMAL(15,2) NOT NULL,
    change_amount DECIMAL(15,2) NOT NULL, -- +/- değişim miktarı
    reason VARCHAR(50) NOT NULL, -- 'credit', 'debit', 'transfer_in', 'transfer_out'
    transaction_id INTEGER REFERENCES transactions(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Performance için index'ler
CREATE INDEX idx_balance_history_user_id ON balance_history(user_id);
CREATE INDEX idx_balance_history_created_at ON balance_history(created_at);
CREATE INDEX idx_balance_history_transaction_id ON balance_history(transaction_id);

-- Composite index for user + date queries
CREATE INDEX idx_balance_history_user_date ON balance_history(user_id, created_at DESC);