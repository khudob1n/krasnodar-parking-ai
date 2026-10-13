package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// FlashCall входит по звонку через Плюсофон (https://plusofon.ru/products/flashcall).
// Плюсофон звонит на номер и сразу сбрасывает. Код - последние 4 цифры
// номера, с которого звонили; его возвращает API, и его же вводит пользователь.
// Для пользователя звонок бесплатный, для нас до 100 звонков в месяц тоже.
//
// Документация: https://help.plusofon.ru/api/v1/flash-call. Адрес метода и
// поля ответа надо сверить с ней при подключении: страница открывается
// только в браузере, и при написании кода прочитать её не удалось.
type FlashCall struct {
	Token   string       // API-токен из личного кабинета Плюсофона
	BaseURL string       // пусто - боевой адрес; в тестах подменяем
	HTTP    *http.Client // пусто - клиент с таймаутом 10 секунд
}

const (
	plusofonURL = "https://restapi.plusofon.ru"
	// plusofonClient - постоянный идентификатор для заголовка Client,
	// его требует каждый запрос к API Плюсофона.
	plusofonClient = "10553"
)

func (f FlashCall) SendCode(ctx context.Context, phone string) (string, error) {
	// Плюсофон ждёт номер без плюса: 79181234567.
	body, err := json.Marshal(map[string]string{"phone": strings.TrimPrefix(phone, "+")})
	if err != nil {
		return "", err
	}
	base := f.BaseURL
	if base == "" {
		base = plusofonURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/api/v1/flash-call/call", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Client", plusofonClient)
	req.Header.Set("Authorization", "Bearer "+f.Token)

	client := f.HTTP
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("plusofon: %w", err)
	}
	defer resp.Body.Close()
	// Читаем не больше 64 КБ: ответ маленький, а лишнее нам не нужно.
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return "", fmt.Errorf("plusofon: read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("plusofon: status %d: %s", resp.StatusCode, raw)
	}

	var out struct {
		Data struct {
			Pin string `json:"pin"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("plusofon: decode response: %w", err)
	}
	if len(out.Data.Pin) != codeDigits {
		return "", fmt.Errorf("plusofon: unexpected pin in response: %s", raw)
	}
	return out.Data.Pin, nil
}
