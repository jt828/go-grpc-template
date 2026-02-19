CREATE TABLE IF NOT EXISTS main.idempotency_records (
    id BIGINT PRIMARY KEY,
    request_type VARCHAR(255) NOT NULL,
    reference_id BIGINT NOT NULL,
    response_data TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
