-- +goose Up
-- Таблицы тестового платёжного провайдера fakepay. Это «база другой
-- компании»: наш код ходит в неё только через fakepay, а не напрямую.
-- С настоящим провайдером этих таблиц не будет.
CREATE TABLE fakepay_cards (
    token      text PRIMARY KEY,
    last4      text NOT NULL,
    brand      text NOT NULL,
    exp_month  integer NOT NULL,
    exp_year   integer NOT NULL,
    behaviour  text NOT NULL CHECK (behaviour IN ('ok', 'declined', 'insufficient')),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE fakepay_charges (
    ref        text PRIMARY KEY,
    token      text   NOT NULL REFERENCES fakepay_cards (token),
    amount     bigint NOT NULL CHECK (amount > 0),
    refunded   bigint NOT NULL DEFAULT 0 CHECK (refunded >= 0 AND refunded <= amount),
    idem_key   text   NOT NULL UNIQUE, -- ключ идемпотентности от клиента провайдера
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Карты пользователя. Номера карт у нас нет, только токен провайдера.
CREATE TABLE cards (
    id         bigserial PRIMARY KEY,
    user_id    bigint  NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token      text    NOT NULL UNIQUE,
    last4      text    NOT NULL,
    brand      text    NOT NULL,
    exp_month  integer NOT NULL,
    exp_year   integer NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX cards_user_idx ON cards (user_id, created_at);

-- Какой картой платится парковка и сколько по ней уже списано за вычетом возвратов.
ALTER TABLE parking_sessions
    ADD COLUMN card_id bigint REFERENCES cards (id) ON DELETE SET NULL,
    ADD COLUMN paid    bigint NOT NULL DEFAULT 0;

-- Каждое движение денег: списание или возврат.
CREATE TABLE payments (
    id           bigserial PRIMARY KEY,
    user_id      bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    session_id   bigint NOT NULL REFERENCES parking_sessions (id) ON DELETE CASCADE,
    card_id      bigint REFERENCES cards (id) ON DELETE SET NULL,
    kind         text   NOT NULL CHECK (kind IN ('charge', 'refund')),
    amount       bigint NOT NULL CHECK (amount > 0),
    provider_ref text   NOT NULL, -- у возврата - ref того списания, которое возвращаем
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX payments_session_idx ON payments (session_id);

-- +goose Down
DROP TABLE payments;
ALTER TABLE parking_sessions DROP COLUMN card_id, DROP COLUMN paid;
DROP TABLE cards;
DROP TABLE fakepay_charges;
DROP TABLE fakepay_cards;
