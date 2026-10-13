package parking

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/payments"
)

// Провайдера зовём внутри транзакции парковки, последним шагом перед
// коммитом: если списание не прошло, откатывается и парковка. Обратная
// сторона: если провайдер списал, а коммит упал, деньги взяты без записи
// у нас. С fakepay в той же базе это почти невозможно; с настоящим
// удалённым провайдером нужна была бы outbox-таблица и повтор запроса с
// тем же ключом идемпотентности.

// pickCard возвращает карту пользователя: cardID или, если 0, последнюю
// добавленную.
func pickCard(ctx context.Context, tx pgx.Tx, userID, cardID int64) (int64, error) {
	if cardID == 0 {
		err := tx.QueryRow(ctx, `
SELECT id FROM cards WHERE user_id = $1 ORDER BY created_at DESC, id DESC LIMIT 1`, userID).Scan(&cardID)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNoCard
		}
		return cardID, err
	}
	var id int64
	err := tx.QueryRow(ctx, `SELECT id FROM cards WHERE id = $1 AND user_id = $2`, cardID, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, payments.ErrCardNotFound
	}
	return id, err
}

// charge списывает amount с карты в счёт парковки и записывает платёж.
// amount <= 0 - ничего не делает.
func (s *Service) charge(ctx context.Context, tx pgx.Tx, userID, sessionID, cardID, amount int64, key string) error {
	if amount <= 0 {
		return nil
	}
	var token string
	err := tx.QueryRow(ctx, `SELECT token FROM cards WHERE id = $1`, cardID).Scan(&token)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNoCard
	}
	if err != nil {
		return fmt.Errorf("load card: %w", err)
	}
	ref, err := s.provider.Charge(ctx, token, amount, key)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
INSERT INTO payments (user_id, session_id, card_id, kind, amount, provider_ref)
VALUES ($1, $2, $3, 'charge', $4, $5)`, userID, sessionID, cardID, amount, ref)
	if err != nil {
		return fmt.Errorf("save payment: %w", err)
	}
	_, err = tx.Exec(ctx, `UPDATE parking_sessions SET paid = paid + $1 WHERE id = $2`, amount, sessionID)
	return err
}

// refund возвращает amount по списаниям парковки, начиная с последних:
// сначала доплату за продление, потом списание при старте.
func (s *Service) refund(ctx context.Context, tx pgx.Tx, userID, sessionID, amount int64) error {
	rows, err := tx.Query(ctx, `
SELECT c.provider_ref, c.card_id,
       c.amount - COALESCE((SELECT sum(r.amount) FROM payments r
                            WHERE r.session_id = c.session_id AND r.kind = 'refund'
                              AND r.provider_ref = c.provider_ref), 0)
FROM payments c
WHERE c.session_id = $1 AND c.kind = 'charge'
ORDER BY c.id DESC`, sessionID)
	if err != nil {
		return fmt.Errorf("load charges: %w", err)
	}
	type charge struct {
		ref    string
		cardID *int64
		left   int64
	}
	charges, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (charge, error) {
		var c charge
		err := row.Scan(&c.ref, &c.cardID, &c.left)
		return c, err
	})
	if err != nil {
		return fmt.Errorf("load charges: %w", err)
	}

	left := amount
	for _, c := range charges {
		part := min(left, c.left)
		if part <= 0 {
			continue
		}
		if err := s.provider.Refund(ctx, c.ref, part); err != nil {
			return fmt.Errorf("refund %s: %w", c.ref, err)
		}
		_, err := tx.Exec(ctx, `
INSERT INTO payments (user_id, session_id, card_id, kind, amount, provider_ref)
VALUES ($1, $2, $3, 'refund', $4, $5)`, userID, sessionID, c.cardID, part, c.ref)
		if err != nil {
			return fmt.Errorf("save refund: %w", err)
		}
		left -= part
		if left == 0 {
			break
		}
	}
	_, err = tx.Exec(ctx, `UPDATE parking_sessions SET paid = paid - $1 WHERE id = $2`, amount-left, sessionID)
	return err
}
