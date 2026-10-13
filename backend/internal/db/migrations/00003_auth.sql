-- +goose Up
CREATE TABLE users (
    id         bigserial PRIMARY KEY,
    phone      text NOT NULL UNIQUE,          -- +7XXXXXXXXXX
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Последний выданный SMS-код на номер. Храним хеш, а не сам код.
CREATE TABLE login_codes (
    phone      text PRIMARY KEY,
    code_hash  bytea NOT NULL,
    sent_at    timestamptz NOT NULL,
    expires_at timestamptz NOT NULL,
    attempts   integer NOT NULL DEFAULT 0
);

-- Refresh-токены тоже храним хешами: утечка таблицы не даст войти.
CREATE TABLE refresh_tokens (
    token_hash bytea PRIMARY KEY,
    user_id    bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX refresh_tokens_user_idx ON refresh_tokens (user_id);

-- +goose Down
DROP TABLE refresh_tokens;
DROP TABLE login_codes;
DROP TABLE users;
