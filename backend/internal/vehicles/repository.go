package vehicles

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound  = errors.New("vehicle not found")
	ErrDuplicate = errors.New("vehicle with this plate already added")
	ErrBadTitle  = errors.New("title must be at most 50 characters")
)

const maxTitleLen = 50

type Vehicle struct {
	ID        int64     `json:"id"`
	Plate     string    `json:"plate"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
}

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

// Add добавляет машину пользователю. Номер и название проверяются здесь,
// чтобы в базу не попало ничего кривого, откуда бы ни пришёл вызов.
func (r *Repository) Add(ctx context.Context, userID int64, plate, title string) (Vehicle, error) {
	plate, err := NormalizePlate(plate)
	if err != nil {
		return Vehicle{}, err
	}
	title = strings.TrimSpace(title)
	if len([]rune(title)) > maxTitleLen {
		return Vehicle{}, ErrBadTitle
	}

	v := Vehicle{Plate: plate, Title: title}
	err = r.pool.QueryRow(ctx,
		`INSERT INTO vehicles (user_id, plate, title) VALUES ($1, $2, $3) RETURNING id, created_at`,
		userID, plate, title,
	).Scan(&v.ID, &v.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return Vehicle{}, ErrDuplicate
	}
	if err != nil {
		return Vehicle{}, fmt.Errorf("add vehicle: %w", err)
	}
	return v, nil
}

func (r *Repository) List(ctx context.Context, userID int64) ([]Vehicle, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, plate, title, created_at FROM vehicles WHERE user_id = $1 ORDER BY created_at, id`, userID)
	if err != nil {
		return nil, fmt.Errorf("list vehicles: %w", err)
	}
	list, err := pgx.CollectRows(rows, pgx.RowToStructByPos[Vehicle])
	if err != nil {
		return nil, fmt.Errorf("list vehicles: %w", err)
	}
	return list, nil
}

// Get возвращает машину, только если она принадлежит этому пользователю.
func (r *Repository) Get(ctx context.Context, userID, id int64) (Vehicle, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, plate, title, created_at FROM vehicles WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return Vehicle{}, fmt.Errorf("get vehicle: %w", err)
	}
	v, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[Vehicle])
	if errors.Is(err, pgx.ErrNoRows) {
		return Vehicle{}, ErrNotFound
	}
	if err != nil {
		return Vehicle{}, fmt.Errorf("get vehicle: %w", err)
	}
	return v, nil
}

func (r *Repository) Delete(ctx context.Context, userID, id int64) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM vehicles WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return fmt.Errorf("delete vehicle: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
