-- +goose Up
-- shape - то, до чего меряем расстояние: участки улицы, а если их нет, точка
-- зоны. Колонка считается сама при каждой записи строки.
ALTER TABLE zones
    ADD COLUMN shape geography GENERATED ALWAYS AS (COALESCE(segments::geography, location::geography)) STORED;

CREATE INDEX zones_shape_idx ON zones USING gist (shape);

-- +goose Down
ALTER TABLE zones DROP COLUMN shape;
