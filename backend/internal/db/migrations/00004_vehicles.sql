-- +goose Up
CREATE TABLE vehicles (
    id         bigserial PRIMARY KEY,
    user_id    bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    plate      text   NOT NULL,          -- А123ВС93, кириллица, без пробелов
    title      text   NOT NULL DEFAULT '', -- как пользователь назвал машину
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, plate)
);

-- +goose Down
DROP TABLE vehicles;
