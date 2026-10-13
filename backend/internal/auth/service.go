// Package auth - вход по номеру телефона и одноразовому коду.
//
// Схема: клиент просит код (RequestCode), мы доставляем его на номер
// (звонком или в лог при разработке). Клиент
// присылает код (Verify) и получает пару токенов: короткий access (JWT,
// им подписывается каждый запрос) и долгий refresh (им получают новую
// пару, когда access истёк).
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"math/big"
	"time"
)

var (
	ErrCodeInvalid     = errors.New("code is invalid or expired")
	ErrTooManyAttempts = errors.New("too many attempts, request a new code")
	ErrNotFound        = errors.New("not found")
)

// TooSoonError - новый код просят раньше, чем можно.
type TooSoonError struct{ RetryAfter time.Duration }

func (e TooSoonError) Error() string {
	return fmt.Sprintf("code already sent, retry in %s", e.RetryAfter.Round(time.Second))
}

const (
	codeTTL     = 5 * time.Minute
	resendAfter = time.Minute
	maxAttempts = 5
	refreshTTL  = 30 * 24 * time.Hour
	codeDigits  = 4
)

type User struct {
	ID        int64     `json:"id"`
	Phone     string    `json:"phone"`
	CreatedAt time.Time `json:"createdAt"`
}

type LoginCode struct {
	Phone     string
	CodeHash  []byte
	SentAt    time.Time
	ExpiresAt time.Time
	Attempts  int
}

// Session - то, что получает клиент после входа.
type Session struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"` // секунд до истечения access-токена
	User         User   `json:"user"`
}

type Store interface {
	SaveCode(ctx context.Context, c LoginCode) error
	GetCode(ctx context.Context, phone string) (LoginCode, error)
	IncrementAttempts(ctx context.Context, phone string) error
	DeleteCode(ctx context.Context, phone string) error
	UpsertUser(ctx context.Context, phone string) (User, error)
	GetUser(ctx context.Context, id int64) (User, error)
	SaveRefresh(ctx context.Context, tokenHash []byte, userID int64, expiresAt time.Time) error
	// TakeRefresh удаляет токен и возвращает его владельца. Токен
	// одноразовый: при обновлении выдаём новый, старый больше не работает.
	TakeRefresh(ctx context.Context, tokenHash []byte, now time.Time) (int64, error)
	DeleteRefresh(ctx context.Context, tokenHash []byte) error
}

// CodeSender доставляет код входа на номер и возвращает этот код.
// Код придумывает сам отправитель: при входе по звонку (FlashCall) это
// последние цифры номера, с которого позвонили, и выбрать их мы не можем.
type CodeSender interface {
	SendCode(ctx context.Context, phone string) (string, error)
}

type Service struct {
	store  Store
	sender CodeSender
	tokens *Tokens
	now    func() time.Time
}

func NewService(store Store, sender CodeSender, tokens *Tokens) *Service {
	return &Service{store: store, sender: sender, tokens: tokens, now: time.Now}
}

// RequestCode создаёт код и отправляет его на номер.
func (s *Service) RequestCode(ctx context.Context, rawPhone string) error {
	phone, err := NormalizePhone(rawPhone)
	if err != nil {
		return err
	}
	now := s.now()

	prev, err := s.store.GetCode(ctx, phone)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	if err == nil {
		if wait := prev.SentAt.Add(resendAfter).Sub(now); wait > 0 {
			return TooSoonError{RetryAfter: wait}
		}
	}

	// Сначала доставляем, потом сохраняем: код знает только отправитель.
	// Если доставка не удалась, старый код остаётся в силе.
	code, err := s.sender.SendCode(ctx, phone)
	if err != nil {
		return fmt.Errorf("send code: %w", err)
	}
	return s.store.SaveCode(ctx, LoginCode{
		Phone:     phone,
		CodeHash:  hash(phone + ":" + code),
		SentAt:    now,
		ExpiresAt: now.Add(codeTTL),
	})
}

// Verify проверяет код и, если он верный, входит (или регистрирует).
func (s *Service) Verify(ctx context.Context, rawPhone, code string) (Session, error) {
	phone, err := NormalizePhone(rawPhone)
	if err != nil {
		return Session{}, err
	}
	saved, err := s.store.GetCode(ctx, phone)
	if errors.Is(err, ErrNotFound) {
		return Session{}, ErrCodeInvalid
	}
	if err != nil {
		return Session{}, err
	}
	if s.now().After(saved.ExpiresAt) {
		return Session{}, ErrCodeInvalid
	}
	if saved.Attempts >= maxAttempts {
		return Session{}, ErrTooManyAttempts
	}
	// ConstantTimeCompare сравнивает за одинаковое время, чтобы по скорости
	// ответа нельзя было угадывать код.
	if subtle.ConstantTimeCompare(saved.CodeHash, hash(phone+":"+code)) != 1 {
		if err := s.store.IncrementAttempts(ctx, phone); err != nil {
			return Session{}, err
		}
		return Session{}, ErrCodeInvalid
	}

	if err := s.store.DeleteCode(ctx, phone); err != nil {
		return Session{}, err
	}
	user, err := s.store.UpsertUser(ctx, phone)
	if err != nil {
		return Session{}, err
	}
	return s.newSession(ctx, user)
}

// Refresh меняет refresh-токен на новую пару токенов.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (Session, error) {
	userID, err := s.store.TakeRefresh(ctx, hash(refreshToken), s.now())
	if errors.Is(err, ErrNotFound) {
		return Session{}, ErrBadToken
	}
	if err != nil {
		return Session{}, err
	}
	user, err := s.store.GetUser(ctx, userID)
	if err != nil {
		return Session{}, err
	}
	return s.newSession(ctx, user)
}

// Logout отзывает refresh-токен. Access-токен доживёт свои минуты сам.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	return s.store.DeleteRefresh(ctx, hash(refreshToken))
}

// Authenticate проверяет access-токен и возвращает id пользователя.
func (s *Service) Authenticate(accessToken string) (int64, error) {
	return s.tokens.Parse(accessToken, s.now())
}

func (s *Service) User(ctx context.Context, id int64) (User, error) {
	return s.store.GetUser(ctx, id)
}

func (s *Service) newSession(ctx context.Context, user User) (Session, error) {
	now := s.now()
	access, err := s.tokens.Issue(user.ID, now)
	if err != nil {
		return Session{}, err
	}
	refresh, err := randomToken()
	if err != nil {
		return Session{}, err
	}
	if err := s.store.SaveRefresh(ctx, hash(refresh), user.ID, now.Add(refreshTTL)); err != nil {
		return Session{}, err
	}
	return Session{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(s.tokens.ttl / time.Second),
		User:         user,
	}, nil
}

// randomCode - случайный код из codeDigits цифр, например "0472".
func randomCode() (string, error) {
	max := big.NewInt(1)
	for range codeDigits {
		max.Mul(max, big.NewInt(10))
	}
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", fmt.Errorf("random code: %w", err)
	}
	return fmt.Sprintf("%0*d", codeDigits, n.Int64()), nil
}
