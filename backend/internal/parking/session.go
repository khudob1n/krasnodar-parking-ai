// Package parking - парковочные сессии: начать, продлить, остановить.
//
// Модель: водитель выбирает зону, машину и сколько минут хочет стоять.
// Если уехал раньше и нажал «Остановить», платит только за фактические
// минуты. Если не остановил, парковка сама заканчивается в planned_end.
//
// Оплата картой, два режима. Предоплата: при старте списываем стоимость
// до planned_end, при продлении - доплату, при остановке возвращаем
// лишнее. Постоплата: время не выбирают, итог списываем при остановке;
// не списалось - остаётся долг, и новую парковку не начать, пока его не
// оплатят.
package parking

import (
	"errors"
	"time"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/pricing"
)

var (
	ErrNotFound        = errors.New("session not found")
	ErrZoneNotFound    = errors.New("zone not found")
	ErrVehicleNotFound = errors.New("vehicle not found")
	ErrFreeZone        = errors.New("parking in this zone is free")
	ErrAlreadyParked   = errors.New("this vehicle already has an active parking")
	ErrNotActive       = errors.New("session is already finished")
	ErrBadDuration     = errors.New("minutes must be between 1 and 1440")
	ErrTooLong         = errors.New("parking can't be longer than 24 hours")
	ErrNoCard          = errors.New("add a payment card first")
	ErrBadMode         = errors.New("unknown parking mode")
	ErrHasDebt         = errors.New("pay for the previous parking first")
	ErrNoEndTime       = errors.New("postpaid parking has no end time")
	ErrNothingToPay    = errors.New("nothing to pay")
)

const maxSession = 24 * time.Hour

type Status string

const (
	StatusActive   Status = "active"
	StatusFinished Status = "finished"
)

// Mode - как платится парковка.
type Mode string

const (
	// ModePrepaid - вперёд за выбранное время, лишнее возвращается при остановке.
	ModePrepaid Mode = "prepaid"
	// ModePostpaid - по факту при остановке. planned_end - только
	// предел в 24 часа, тогда парковка закроется сама.
	ModePostpaid Mode = "postpaid"
)

type Session struct {
	ID         int64          `json:"id"`
	Zone       int            `json:"zone"`
	Plate      string         `json:"plate"`
	Tariff     pricing.Tariff `json:"tariff"`
	StartedAt  time.Time      `json:"startedAt"`
	PlannedEnd time.Time      `json:"plannedEnd"`
	EndedAt    *time.Time     `json:"endedAt"` // nil, пока активна
	Status     Status         `json:"status"`
	Mode       Mode           `json:"mode"`
	// Cost - итог в копейках для завершённой; для активной - сколько
	// выйдет, если стоять до plannedEnd (постоплатной - сколько набежало
	// к этому моменту).
	Cost int64 `json:"cost"`
	// Paid - сколько списано с карты за вычетом возвратов, копейки.
	Paid   int64  `json:"paid"`
	CardID *int64 `json:"cardId"` // nil, если карту удалили
	// Debt - сколько ещё должны: только у завершённой постоплатной,
	// если итог не списался.
	Debt         int64  `json:"debt"`
	PaymentError string `json:"paymentError,omitempty"` // почему не списалось
}

// finalize заполняет Status и Cost по состоянию на момент now.
// Сессию, у которой прошло оплаченное время, показываем завершённой,
// даже если в базе её ещё не закрыли.
func (s *Session) finalize(now time.Time, storedCost *int64) error {
	if s.EndedAt == nil && !now.Before(s.PlannedEnd) {
		end := s.PlannedEnd
		s.EndedAt = &end
	}
	if s.EndedAt == nil {
		s.Status = StatusActive
		end := s.PlannedEnd
		if s.Mode == ModePostpaid {
			if !now.After(s.StartedAt) {
				s.Cost = 0
				return nil
			}
			end = now
		}
		q, err := pricing.Calculate(s.Tariff, s.StartedAt, end)
		s.Cost = q.Total
		return err
	}
	s.Status = StatusFinished
	if storedCost != nil {
		s.Cost = *storedCost
		return nil
	}
	if !s.EndedAt.After(s.StartedAt) { // остановили в ту же секунду
		s.Cost = 0
		return nil
	}
	q, err := pricing.Calculate(s.Tariff, s.StartedAt, *s.EndedAt)
	s.Cost = q.Total
	return err
}
