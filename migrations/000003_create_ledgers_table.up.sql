CREATE TABLE IF NOT EXISTS main.ledgers (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    transaction_type VARCHAR(16) NOT NULL,
    token VARCHAR(32) NOT NULL,
    amount NUMERIC(36, 18) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
