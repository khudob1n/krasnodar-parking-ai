-- +goose Up
-- Токены push-уведомлений устройств пользователя.
CREATE TABLE push_tokens (
    token      text PRIMARY KEY,
    user_id    bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    platform   text   NOT NULL CHECK (platform IN ('android', 'ios', 'rustore')),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX push_tokens_user_idx ON push_tokens (user_id);

-- Когда отправили «скоро закончится» и «время вышло». NULL - ещё не отправляли.
ALTER TABLE parking_sessions
    ADD COLUMN reminded_at         timestamptz,
    ADD COLUMN expired_notified_at timestamptz;

-- +goose Down
ALTER TABLE parking_sessions DROP COLUMN reminded_at, DROP COLUMN expired_notified_at;
DROP TABLE push_tokens;
