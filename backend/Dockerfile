# Собираем из корня репозитория: docker build -f backend/Dockerfile .
# Корень нужен, чтобы положить в образ data/zones.json для импорта.

# 1. Сборка. В этом образе есть Go, но в итоговый он не попадёт.
FROM golang:1.26 AS build
WORKDIR /src
# Сначала только go.mod/go.sum: слой с зависимостями кешируется и не
# перекачивается, пока они не поменялись.
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# CGO_ENABLED=0 - статический бинарник, которому не нужны системные библиотеки.
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/import-zones ./cmd/import-zones

# 2. Итоговый образ: только бинарники, без шелла и пакетного менеджера.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /out/import-zones /
COPY data/zones.json /data/zones.json
USER nonroot
EXPOSE 8080
ENTRYPOINT ["/api"]
