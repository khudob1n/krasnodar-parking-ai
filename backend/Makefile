DB_URL ?= postgres://parking:parking@localhost:5432/parking?sslmode=disable

.PHONY: db-up db-down import run test

db-up:
	docker compose -f ../docker-compose.yml up -d --wait

db-down:
	docker compose -f ../docker-compose.yml down

import:
	DATABASE_URL=$(DB_URL) go run ./cmd/import-zones -file ../data/zones.json

run:
	DATABASE_URL=$(DB_URL) JWT_SECRET=dev-secret-dev-secret-dev-secret-00 go run ./cmd/api

test:
	TEST_DATABASE_URL=$(DB_URL) go test -p 1 ./...
