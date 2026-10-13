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
	// ZonesFile - выгрузка зон, которую API загрузит при старте, если таблица зон пуста.
	// Нужна бесплатным хостингам, где нельзя отдельно запустить import-zones.
	ZonesFile string
	// UCallerServiceID и UCallerKey - доступ к uCaller для входа по звонку.
	// Пусто - код не отправляется, а пишется в лог (режим разработки).
	UCallerServiceID string
	UCallerKey       string
}

func Load() (Config, error) {
	cfg := Config{
		DatabaseURL:      getenv("DATABASE_URL", "postgres://parking:parking@localhost:5432/parking?sslmode=disable"),
		HTTPAddr:         getenv("HTTP_ADDR", defaultAddr()),
		JWTSecret:        os.Getenv("JWT_SECRET"),
		AccessTokenTTL:   15 * time.Minute,
		ZonesFile:        os.Getenv("IMPORT_ZONES_FILE"),
		UCallerServiceID: os.Getenv("UCALLER_SERVICE_ID"),
		UCallerKey:       os.Getenv("UCALLER_KEY"),
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

// defaultAddr - порт из PORT, если хостинг его задаёт (Render, Koyeb, Railway), иначе 8080.
func defaultAddr() string {
	if port := os.Getenv("PORT"); port != "" {
		return ":" + port
	}
	return ":8080"
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
