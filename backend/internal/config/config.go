// Package config читает настройки из переменных окружения.
package config

import (
	"errors"
	"os"
	"time"
)

type Config struct {
	DatabaseURL    string        // строка подключения к Postgres
	HTTPAddr       string        // адрес, на котором слушает API
	JWTSecret      string        // секрет для подписи access-токенов
	AccessTokenTTL time.Duration // сколько живёт access-токен
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:    getenv("DATABASE_URL", "postgres://parking:parking@localhost:5432/parking?sslmode=disable"),
		HTTPAddr:       getenv("HTTP_ADDR", ":8080"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		AccessTokenTTL: 15 * time.Minute,
	}
	if raw := os.Getenv("ACCESS_TOKEN_TTL"); raw != "" {
		ttl, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, errors.New("ACCESS_TOKEN_TTL must be a duration like 15m")
		}
		cfg.AccessTokenTTL = ttl
	}
	return cfg, nil
}

// RequireJWTSecret нужен только API: импортёру зон секрет ни к чему.
func (c Config) RequireJWTSecret() error {
	if len(c.JWTSecret) < 32 {
		return errors.New("JWT_SECRET must be set and at least 32 characters long")
	}
	return nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
