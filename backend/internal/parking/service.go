package parking

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/payments"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/pricing"
)

type Service struct {
	pool     *pgxpool.Pool
	provider payments.Provider
	now      func() time.Time
}

func NewService(pool *pgxpool.Pool, provider payments.Provider) *Service {
	return &Service{pool: pool, provider: provider, now: time.Now}
}

// StartRequest - что прислал водитель.
type StartRequest struct {
	Zone      int   `json:"zone"`
	VehicleID int64 `json:"vehicleId"`
	Minutes   int   `json:"minutes"`
	CardID    int64 `json:"cardId"` // 0 - последняя добавленная карта
	Mode      Mode  `json:"mode"`   // пусто - prepaid
}

// Start начинает парковку. Всё в одной транзакции, чтобы между проверкой
// «машина нигде не стоит» и вставкой никто не вклинился. Если списать
// деньги не вышло, парковка не создаётся.
func (s *Service) Start(ctx context.Context, userID int64, req StartRequest) (Session, error) {
	if req.Mode != "" && req.Mode != ModePrepaid {
		return Session{}, ErrBadMode
	}
	if req.Minutes < 1 || req.Minutes > int(maxSession/time.Minute) {
		return Session{}, ErrBadDuration
	}
	now := s.now().Truncate(time.Second)

	var id int64
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var plate string
		err := tx.QueryRow(ctx, `SELECT plate FROM vehicles WHERE id = $1 AND user_id = $2`,
			req.VehicleID, userID).Scan(&plate)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrVehicleNotFound
		}
		if err != nil {
			return err
		}

		var kind string
		var price int64
		err = tx.QueryRow(ctx, `SELECT kind, price_per_hour FROM zones WHERE number = $1`, req.Zone).
			Scan(&kind, &price)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrZoneNotFound
		}
		if err != nil {
			return err
		}
		if kind == "free" || price == 0 {
			return ErrFreeZone
		}

		// Если у машины висит парковка, чьё время уже вышло, закрываем её.
		if _, err := closeExpired(ctx, tx, now, plate); err != nil {
			return err
		}

		cardID, err := pickCard(ctx, tx, userID, req.CardID)
		if err != nil {
			return err
		}

		end := now.Add(time.Duration(req.Minutes) * time.Minute)
		var tariff pricing.Tariff
		err = tx.QueryRow(ctx, `
INSERT INTO parking_sessions
    (user_id, zone_number, plate, price_per_hour, paid_from, paid_to, started_at, planned_end, card_id)
SELECT $1, number, $2, price_per_hour, paid_from, paid_to, $3, $4, $6
FROM zones WHERE number = $5
RETURNING id, price_per_hour, to_char(paid_from, 'HH24:MI'), to_char(paid_to, 'HH24:MI')`,
			userID, plate, now, end, req.Zone, cardID,
		).Scan(&id, &tariff.PricePerHour, &tariff.PaidFrom, &tariff.PaidTo)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrAlreadyParked
		}
		if err != nil {
			return err
		}

		// Списываем вперёд всё время до planned_end.
		q, err := pricing.Calculate(tariff, now, end)
		if err != nil {
			return err
		}
		return s.charge(ctx, tx, userID, id, cardID, q.Total, fmt.Sprintf("session-%d-start", id))
	})
	if err != nil {
		return Session{}, err
	}
	return s.Get(ctx, userID, id)
}

// Extend продлевает активную парковку на minutes минут и списывает
// доплату картой парковки. Не списалось - парковка не меняется.
func (s *Service) Extend(ctx context.Context, userID, id int64, minutes int) (Session, error) {
	if minutes < 1 || minutes > int(maxSession/time.Minute) {
		return Session{}, ErrBadDuration
	}
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		sess, err := s.lockActive(ctx, tx, userID, id)
		if err != nil {
			return err
		}
		end := sess.PlannedEnd.Add(time.Duration(minutes) * time.Minute)
		if end.Sub(sess.StartedAt) > maxSession {
			return ErrTooLong
		}
		// reminded_at сбрасываем, чтобы напомнить и о новом конце.
		_, err = tx.Exec(ctx, `UPDATE parking_sessions SET planned_end = $1, reminded_at = NULL WHERE id = $2`, end, id)
		if err != nil {
			return err
		}

		q, err := pricing.Calculate(sess.Tariff, sess.StartedAt, end)
		if err != nil {
			return err
		}
		extra := q.Total - sess.Paid
		if extra <= 0 { // продлили на бесплатные часы
			return nil
		}
		if sess.CardID == nil {
			return ErrNoCard
		}
		// Ключ с новым концом: новый конец всегда позже старого, так что
		// разные продления не склеятся, а повтор того же - склеится.
		return s.charge(ctx, tx, userID, id, *sess.CardID, extra, fmt.Sprintf("session-%d-extend-%d", id, end.Unix()))
	})
	if err != nil {
		return Session{}, err
	}
	return s.Get(ctx, userID, id)
}

// Stop завершает парковку сейчас. Платим только за фактическое время:
// разницу с оплаченным возвращаем на карту.
func (s *Service) Stop(ctx context.Context, userID, id int64) (Session, error) {
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		sess, err := s.lockActive(ctx, tx, userID, id)
		if err != nil {
			return err
		}
		ended := s.now().Truncate(time.Second)
		sess.EndedAt = &ended
		if err := sess.finalize(ended, nil); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE parking_sessions SET ended_at = $1, cost = $2 WHERE id = $3`,
			ended, sess.Cost, id)
		if err != nil {
			return err
		}
		if back := sess.Paid - sess.Cost; back > 0 {
			return s.refund(ctx, tx, userID, id, back)
		}
		return nil
	})
	if err != nil {
		return Session{}, err
	}
	return s.Get(ctx, userID, id)
}

// Get - одна парковка пользователя.
func (s *Service) Get(ctx context.Context, userID, id int64) (Session, error) {
	list, err := s.query(ctx, `WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return Session{}, err
	}
	if len(list) == 0 {
		return Session{}, ErrNotFound
	}
	return list[0], nil
}

// List - последние парковки пользователя, свежие первыми.
func (s *Service) List(ctx context.Context, userID int64, limit int) ([]Session, error) {
	return s.query(ctx, `WHERE user_id = $1 ORDER BY started_at DESC, id DESC LIMIT $2`, userID, limit)
}

// CloseExpired закрывает все парковки, у которых вышло оплаченное время.
// Возвращает, сколько закрыто. Пригодится фоновой задаче.
func (s *Service) CloseExpired(ctx context.Context) (int, error) {
	var n int
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		var err error
		n, err = closeExpired(ctx, tx, s.now(), "")
		return err
	})
	return n, err
}

const sessionColumns = `
    id, zone_number, plate, price_per_hour,
    to_char(paid_from, 'HH24:MI'), to_char(paid_to, 'HH24:MI'),
    started_at, planned_end, ended_at, cost, card_id, paid`

func scanSession(row pgx.Row) (Session, *int64, error) {
	sess := Session{Mode: ModePrepaid}
	var cost *int64
	err := row.Scan(&sess.ID, &sess.Zone, &sess.Plate, &sess.Tariff.PricePerHour,
		&sess.Tariff.PaidFrom, &sess.Tariff.PaidTo,
		&sess.StartedAt, &sess.PlannedEnd, &sess.EndedAt, &cost, &sess.CardID, &sess.Paid)
	return sess, cost, err
}

func (s *Service) query(ctx context.Context, where string, args ...any) ([]Session, error) {
	rows, err := s.pool.Query(ctx, `SELECT`+sessionColumns+` FROM parking_sessions `+where, args...)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	now := s.now()
	list := []Session{}
	for rows.Next() {
		sess, cost, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		if err := sess.finalize(now, cost); err != nil {
			return nil, err
		}
		list = append(list, sess)
	}
	return list, rows.Err()
}

// lockActive читает парковку с блокировкой строки (FOR UPDATE) и
// проверяет, что она ещё идёт.
func (s *Service) lockActive(ctx context.Context, tx pgx.Tx, userID, id int64) (Session, error) {
	sess, _, err := scanSession(tx.QueryRow(ctx,
		`SELECT`+sessionColumns+` FROM parking_sessions WHERE user_id = $1 AND id = $2 FOR UPDATE`, userID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrNotFound
	}
	if err != nil {
		return Session{}, err
	}
	if sess.EndedAt != nil || !s.now().Before(sess.PlannedEnd) {
		return Session{}, ErrNotActive
	}
	return sess, nil
}

// closeExpired проставляет ended_at и cost парковкам, у которых вышло
// время. plate == "" - для всех машин. Возвращает, сколько закрыто.
// Деньги не двигаются: стоимость до planned_end списали вперёд.
func closeExpired(ctx context.Context, tx pgx.Tx, now time.Time, plate string) (int, error) {
	rows, err := tx.Query(ctx, `SELECT`+sessionColumns+` FROM parking_sessions
WHERE ended_at IS NULL AND planned_end <= $1 AND ($2 = '' OR plate = $2)
FOR UPDATE`, now, plate)
	if err != nil {
		return 0, fmt.Errorf("find expired sessions: %w", err)
	}
	expired, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Session, error) {
		sess, _, err := scanSession(row)
		return sess, err
	})
	if err != nil {
		return 0, fmt.Errorf("find expired sessions: %w", err)
	}
	for _, sess := range expired {
		q, err := pricing.Calculate(sess.Tariff, sess.StartedAt, sess.PlannedEnd)
		if err != nil {
			return 0, err
		}
		_, err = tx.Exec(ctx, `UPDATE parking_sessions SET ended_at = planned_end, cost = $1 WHERE id = $2`,
			q.Total, sess.ID)
		if err != nil {
			return 0, fmt.Errorf("close session %d: %w", sess.ID, err)
		}
	}
	return len(expired), nil
}
