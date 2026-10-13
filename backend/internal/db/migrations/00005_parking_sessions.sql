-- +goose Up
CREATE TABLE parking_sessions (
    id             bigserial PRIMARY KEY,
    user_id        bigint  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    zone_number    integer NOT NULL REFERENCES zones (number),
    plate          text    NOT NULL,
    -- Снимок тарифа на момент старта: если тариф зоны поменяется,
    -- уже начатая парковка считается по старому.
    price_per_hour bigint  NOT NULL,
    paid_from      time    NOT NULL,
    paid_to        time    NOT NULL,
    started_at     timestamptz NOT NULL,
    planned_end    timestamptz NOT NULL,  -- до какого времени оплачено
    ended_at       timestamptz,           -- NULL, пока парковка не закрыта
    cost           bigint,                -- копейки, считается при закрытии
    CHECK (planned_end > started_at)
);

-- Одна машина не может стоять на двух парковках сразу.
CREATE UNIQUE INDEX parking_sessions_active_plate_idx ON parking_sessions (plate) WHERE ended_at IS NULL;
CREATE INDEX parking_sessions_user_idx ON parking_sessions (user_id, started_at DESC);

-- +goose Down
DROP TABLE parking_sessions;
