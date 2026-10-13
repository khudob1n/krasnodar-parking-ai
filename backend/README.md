# Бэкенд

Go + Postgres/PostGIS. Миграции лежат в `internal/db/migrations` и применяются
автоматически при старте `api` и `import-zones`.

## Запуск

```sh
cd backend
make db-up     # поднять Postgres с PostGIS в Docker
make import    # загрузить зоны из ../data/zones.json
make run       # API на :8080
make test      # тесты (интеграционные ходят в ту же базу)
```

## API

- `GET /healthz` - жив ли сервер и база
- `GET /api/v1/zones/{number}` - зона по номеру с таблички
- `GET /api/v1/zones/nearby?lat=45.0355&lon=38.9753&radius=500&limit=20&kind=street,lot` -
  зоны рядом, ближайшие первыми. Расстояние считается до ближайшего участка улицы.
  `radius` до 5000 м (по умолчанию 500), `limit` до 100 (по умолчанию 20), `kind` необязателен.
- `GET /api/v1/zones/{number}/price?minutes=90&start=2026-10-13T18:00:00+03:00` -
  сколько стоит стоянка. `start` необязателен (по умолчанию сейчас). Оплата
  поминутная по местному времени, бесплатные часы не оплачиваются, сумма
  округляется вверх до копейки.

### Вход

Вход по номеру телефона и SMS-коду. Пока SMS-шлюза нет, код пишется в лог API
(строка `SMS not sent, dev mode`).

- `POST /api/v1/auth/code` `{"phone": "+7 918 123-45-67"}` - отправить код (не чаще раза в минуту, 429 с `Retry-After`)
- `POST /api/v1/auth/verify` `{"phone": "...", "code": "1234"}` - войти; ответ: `accessToken` (15 минут), `refreshToken` (30 дней), `user`
- `POST /api/v1/auth/refresh` `{"refreshToken": "..."}` - новая пара токенов, старый refresh больше не работает
- `POST /api/v1/auth/logout` `{"refreshToken": "..."}` - выйти
- `GET /api/v1/me` с заголовком `Authorization: Bearer <accessToken>` - текущий пользователь

Код живёт 5 минут, на него 5 попыток. Для API обязательна переменная
`JWT_SECRET` (не короче 32 символов); `make run` подставляет тестовую.

### Автомобили

Все ручки требуют `Authorization: Bearer <accessToken>`.

- `GET /api/v1/vehicles` - мои машины
- `POST /api/v1/vehicles` `{"plate": "а 123 вс 93", "title": "Киа"}` - добавить. Номер
  приводится к виду `А123ВС93`, латиница заменяется кириллицей. 409, если номер уже добавлен.
- `DELETE /api/v1/vehicles/{id}` - удалить

## Данные зон

`parkingForm` из выгрузки сайта переводится в `kind`:
`1` - `street` (вдоль улицы, есть участки), `0` - `lot` (площадка, одна точка),
`2` - `free` (бесплатная). Цена хранится в копейках. `paidFrom` и `paidTo`
равны `00:00` - тариф круглосуточный. Поля `freeCount` и `placeUpdate`
не импортируются: на сайте это заглушки (свободно всегда = всего мест).
