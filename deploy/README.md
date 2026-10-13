# Деплой

Нужен сервер с Docker (подойдёт любой VPS на Ubuntu с 1 ГБ памяти) и домен,
A-запись которого указывает на сервер. Порты 80 и 443 должны быть открыты:
Caddy сам получит HTTPS-сертификат Let's Encrypt.

```sh
git clone https://github.com/khudob1n/krasnodar-parking-ai.git
cd krasnodar-parking-ai/deploy
cp .env.example .env
# впиши DOMAIN и два секрета: openssl rand -hex 32
docker compose up -d --build
# загрузить зоны (и после каждого обновления data/zones.json)
docker compose run --rm --entrypoint /import-zones api -file /data/zones.json
curl https://$DOMAIN/healthz   # {"status":"ok"}
```

Обновление: `git pull && docker compose up -d --build`. Миграции базы
применяются сами при старте API.

Вместо сборки на сервере можно брать готовый образ, который CI публикует
из main: `API_IMAGE=ghcr.io/khudob1n/krasnodar-parking-ai-api:latest docker compose up -d --no-build`
(для приватного репозитория сначала `docker login ghcr.io`).

Бэкап базы: `docker compose exec db pg_dump -U parking parking | gzip > backup.sql.gz`.
