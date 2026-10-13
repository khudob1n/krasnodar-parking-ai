-- +goose Up
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE zones (
    number          integer PRIMARY KEY,           -- номер с таблички
    kind            text    NOT NULL CHECK (kind IN ('street', 'lot', 'free')),
    address         text    NOT NULL,
    description     text    NOT NULL DEFAULT '',
    location        geography(Point, 4326) NOT NULL,
    segments        geography(MultiLineString, 4326),  -- NULL, если участков нет
    price_per_hour  bigint  NOT NULL CHECK (price_per_hour >= 0), -- копейки
    paid_from       time    NOT NULL,
    paid_to         time    NOT NULL,
    total_places    integer NOT NULL,
    disabled_places integer NOT NULL,
    in_building     boolean NOT NULL,
    tariff_text     text    NOT NULL,
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX zones_location_idx ON zones USING gist (location);

-- +goose Down
DROP TABLE zones;
