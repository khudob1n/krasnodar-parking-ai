-- +goose Up
-- Постоплата: время не выбирают, списываем по факту при остановке.
-- Если итог не списался, парковка всё равно завершается, а сумма
-- становится долгом, пока его не оплатят.
ALTER TABLE parking_sessions
    ADD COLUMN mode          text   NOT NULL DEFAULT 'prepaid' CHECK (mode IN ('prepaid', 'postpaid')),
    ADD COLUMN debt          bigint NOT NULL DEFAULT 0 CHECK (debt >= 0),
    ADD COLUMN payment_error text; -- почему не списалось, NULL - если долга нет

-- Долги проверяются при каждом старте парковки.
CREATE INDEX parking_sessions_debt_idx ON parking_sessions (user_id) WHERE debt > 0;

-- +goose Down
DROP INDEX parking_sessions_debt_idx;
ALTER TABLE parking_sessions DROP COLUMN mode, DROP COLUMN debt, DROP COLUMN payment_error;
