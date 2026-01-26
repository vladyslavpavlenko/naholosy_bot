CREATE TABLE IF NOT EXISTS users (
    id         BIGINT       PRIMARY KEY,
    username   TEXT         NOT NULL,
    created_at TIMESTAMP    NOT NULL DEFAULT now(),
    updated_at TIMESTAMP    NOT NULL DEFAULT now()
);