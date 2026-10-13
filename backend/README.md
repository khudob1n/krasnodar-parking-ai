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

## Данные зон

`parkingForm` из выгрузки сайта переводится в `kind`:
`1` - `street` (вдоль улицы, есть участки), `0` - `lot` (площадка, одна точка),
`2` - `free` (бесплатная). Цена хранится в копейках. `paidFrom` и `paidTo`
равны `00:00` - тариф круглосуточный. Поля `freeCount` и `placeUpdate`
не импортируются: на сайте это заглушки (свободно всегда = всего мест).
