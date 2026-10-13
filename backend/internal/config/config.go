// Package config читает настройки из переменных окружения.
package config

import "os"

type Config struct {
	DatabaseURL string // строка подключения к Postgres
	HTTPAddr    string // адрес, на котором слушает API
}

func Load() Config {
	return Config{
		DatabaseURL: getenv("DATABASE_URL", "postgres://parking:parking@localhost:5432/parking?sslmode=disable"),
		HTTPAddr:    getenv("HTTP_ADDR", ":8080"),
	}
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
