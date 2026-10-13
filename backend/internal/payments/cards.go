package payments

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrCardNotFound  = errors.New("card not found")
	ErrDuplicateCard = errors.New("card already added")
)

// UserCard - привязанная карта пользователя. Токен наружу не отдаём.
type UserCard struct {
	ID int64 `json:"id"`
	Card
	CreatedAt time.Time `json:"createdAt"`
}

// Cards - карты пользователей в нашей базе.
type Cards struct {
	pool     *pgxpool.Pool
	provider Provider
}

func NewCards(pool *pgxpool.Pool, provider Provider) *Cards {
	return &Cards{pool: pool, provider: provider}
}

// Add проверяет карту у провайдера и привязывает её к пользователю.
func (c *Cards) Add(ctx context.Context, userID int64, token string) (UserCard, error) {
	card, err := c.provider.Verify(ctx, token)
	if err != nil {
		return UserCard{}, err
	}
	uc := UserCard{Card: card}
	err = c.pool.QueryRow(ctx, `
INSERT INTO cards (user_id, token, last4, brand, exp_month, exp_year)
VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at`,
		userID, token, card.Last4, card.Brand, card.ExpMonth, card.ExpYear,
	).Scan(&uc.ID, &uc.CreatedAt)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return UserCard{}, ErrDuplicateCard
	}
	if err != nil {
		return UserCard{}, fmt.Errorf("add card: %w", err)
	}
	return uc, nil
}

// List - карты пользователя, старые первыми.
func (c *Cards) List(ctx context.Context, userID int64) ([]UserCard, error) {
	rows, err := c.pool.Query(ctx, `
SELECT id, last4, brand, exp_month, exp_year, created_at
FROM cards WHERE user_id = $1 ORDER BY created_at, id`, userID)
	if err != nil {
		return nil, fmt.Errorf("list cards: %w", err)
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (UserCard, error) {
		var uc UserCard
		err := row.Scan(&uc.ID, &uc.Last4, &uc.Brand, &uc.ExpMonth, &uc.ExpYear, &uc.CreatedAt)
		return uc, err
	})
	if err != nil {
		return nil, fmt.Errorf("list cards: %w", err)
	}
	return list, nil
}

// Delete отвязывает карту. У парковок, которые ею платились, card_id
// станет NULL, история платежей остаётся.
func (c *Cards) Delete(ctx context.Context, userID, id int64) error {
	tag, err := c.pool.Exec(ctx, `DELETE FROM cards WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return fmt.Errorf("delete card: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCardNotFound
	}
	return nil
}
