package reminders

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrBadDevice = errors.New("token is required and platform must be android, ios or rustore")

// Devices хранит push-токены устройств.
type Devices struct{ pool *pgxpool.Pool }

func NewDevices(pool *pgxpool.Pool) *Devices { return &Devices{pool: pool} }

// Register привязывает токен к пользователю. Если токен уже был у другого
// пользователя (вышли и вошли под другим номером на том же телефоне),
// он переходит к новому.
func (d *Devices) Register(ctx context.Context, userID int64, token, platform string) error {
	token = strings.TrimSpace(token)
	switch platform {
	case "android", "ios", "rustore":
	default:
		return ErrBadDevice
	}
	if token == "" || len(token) > 4096 {
		return ErrBadDevice
	}
	_, err := d.pool.Exec(ctx, `
INSERT INTO push_tokens (token, user_id, platform) VALUES ($1, $2, $3)
ON CONFLICT (token) DO UPDATE SET user_id = EXCLUDED.user_id, platform = EXCLUDED.platform`,
		token, userID, platform)
	if err != nil {
		return fmt.Errorf("register device: %w", err)
	}
	return nil
}

// Unregister отвязывает токен (например, при выходе из аккаунта).
func (d *Devices) Unregister(ctx context.Context, userID int64, token string) error {
	_, err := d.pool.Exec(ctx, `DELETE FROM push_tokens WHERE token = $1 AND user_id = $2`, token, userID)
	if err != nil {
		return fmt.Errorf("unregister device: %w", err)
	}
	return nil
}
