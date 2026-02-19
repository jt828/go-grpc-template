CREATE SCHEMA IF NOT EXISTS main;

CREATE TABLE IF NOT EXISTS main.idempotency_records (
    id BIGINT PRIMARY KEY,
    request_type VARCHAR(255) NOT NULL,
    reference_id BIGINT NOT NULL,
    response_data TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS main.users (
    id BIGINT PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    username VARCHAR(255) NOT NULL,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
