CREATE TABLE audit_logs (
    id SERIAL PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL, -- 'user', 'transaction', 'balance'
    entity_id INTEGER NOT NULL,       -- ilgili kaydın ID'si
    action VARCHAR(50) NOT NULL,      -- 'create', 'update', 'delete', 'transfer'
    user_id INTEGER REFERENCES users(id), -- işlemi yapan kullanıcı
    old_data JSONB,                   -- önceki veri (update/delete için)
    new_data JSONB,                   -- yeni veri (create/update için)
    details TEXT,                     -- ek açıklama
    ip_address INET,                  -- kullanıcının IP'si
    user_agent TEXT,                  -- browser bilgisi
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Performance ve sorgular için index'ler
CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);