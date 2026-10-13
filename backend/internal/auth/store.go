package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStore - Store поверх Postgres.
type PGStore struct{ pool *pgxpool.Pool }

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

func (s *PGStore) SaveCode(ctx context.Context, c LoginCode) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO login_codes (phone, code_hash, sent_at, expires_at, attempts)
VALUES ($1, $2, $3, $4, 0)
ON CONFLICT (phone) DO UPDATE SET
    code_hash = EXCLUDED.code_hash, sent_at = EXCLUDED.sent_at,
    expires_at = EXCLUDED.expires_at, attempts = 0`,
		c.Phone, c.CodeHash, c.SentAt, c.ExpiresAt)
	if err != nil {
		return fmt.Errorf("save code: %w", err)
	}
	return nil
}

func (s *PGStore) GetCode(ctx context.Context, phone string) (LoginCode, error) {
	c := LoginCode{Phone: phone}
	err := s.pool.QueryRow(ctx,
		`SELECT code_hash, sent_at, expires_at, attempts FROM login_codes WHERE phone = $1`, phone,
	).Scan(&c.CodeHash, &c.SentAt, &c.ExpiresAt, &c.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return LoginCode{}, ErrNotFound
	}
	if err != nil {
		return LoginCode{}, fmt.Errorf("get code: %w", err)
	}
	return c, nil
}

func (s *PGStore) IncrementAttempts(ctx context.Context, phone string) error {
	_, err := s.pool.Exec(ctx, `UPDATE login_codes SET attempts = attempts + 1 WHERE phone = $1`, phone)
	if err != nil {
		return fmt.Errorf("increment attempts: %w", err)
	}
	return nil
}

func (s *PGStore) DeleteCode(ctx context.Context, phone string) error {
	if _, err := s.pool.Exec(ctx, `DELETE FROM login_codes WHERE phone = $1`, phone); err != nil {
		return fmt.Errorf("delete code: %w", err)
	}
	return nil
}

func (s *PGStore) UpsertUser(ctx context.Context, phone string) (User, error) {
	// DO UPDATE с тем же значением нужен, чтобы RETURNING вернул строку
	// и для уже существующего пользователя.
	u := User{Phone: phone}
	err := s.pool.QueryRow(ctx, `
INSERT INTO users (phone) VALUES ($1)
ON CONFLICT (phone) DO UPDATE SET phone = EXCLUDED.phone
RETURNING id, created_at`, phone).Scan(&u.ID, &u.CreatedAt)
	if err != nil {
		return User{}, fmt.Errorf("upsert user: %w", err)
	}
	return u, nil
}

func (s *PGStore) GetUser(ctx context.Context, id int64) (User, error) {
	u := User{ID: id}
	err := s.pool.QueryRow(ctx, `SELECT phone, created_at FROM users WHERE id = $1`, id).
		Scan(&u.Phone, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("get user: %w", err)
	}
	return u, nil
}

func (s *PGStore) SaveRefresh(ctx context.Context, tokenHash []byte, userID int64, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt)
	if err != nil {
		return fmt.Errorf("save refresh token: %w", err)
	}
	return nil
}

func (s *PGStore) TakeRefresh(ctx context.Context, tokenHash []byte, now time.Time) (int64, error) {
	var userID int64
	var expiresAt time.Time
	err := s.pool.QueryRow(ctx,
		`DELETE FROM refresh_tokens WHERE token_hash = $1 RETURNING user_id, expires_at`, tokenHash,
	).Scan(&userID, &expiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("take refresh token: %w", err)
	}
	if now.After(expiresAt) {
		return 0, ErrNotFound
	}
	return userID, nil
}

func (s *PGStore) DeleteRefresh(ctx context.Context, tokenHash []byte) error {
	if _, err := s.pool.Exec(ctx, `DELETE FROM refresh_tokens WHERE token_hash = $1`, tokenHash); err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}
	return nil
}
