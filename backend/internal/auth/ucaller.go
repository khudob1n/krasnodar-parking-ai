package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// UCaller входит по звонку через uCaller (https://ucaller.ru).
// uCaller звонит на номер и сразу сбрасывает. Мы сами придумываем код,
// и звонок приходит с номера, который на этот код заканчивается:
// пользователь вводит последние 4 цифры. Для пользователя звонок бесплатный.
//
// Документация метода: https://developer.ucaller.ru/methods/initcall/
type UCaller struct {
	ServiceID string       // id сервиса из раздела «Мои сервисы» в кабинете uCaller
	Key       string       // секретный ключ этого сервиса
	BaseURL   string       // пусто - боевой адрес; в тестах подменяем
	HTTP      *http.Client // пусто - клиент с таймаутом 10 секунд
}

const ucallerURL = "https://api.ucaller.ru"

func (u UCaller) SendCode(ctx context.Context, phone string) (string, error) {
	code, err := randomCode()
	if err != nil {
		return "", err
	}
	// uCaller ждёт номер числом без плюса: 79181234567.
	number, err := strconv.ParseInt(strings.TrimPrefix(phone, "+"), 10, 64)
	if err != nil {
		return "", fmt.Errorf("ucaller: phone %q: %w", phone, err)
	}
	body, err := json.Marshal(map[string]any{"phone": number, "code": code})
	if err != nil {
		return "", err
	}
	base := u.BaseURL
	if base == "" {
		base = ucallerURL
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/v1.0/initCall", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+u.Key+"."+u.ServiceID)

	client := u.HTTP
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ucaller: %w", err)
	}
	defer resp.Body.Close()
	// Читаем не больше 64 КБ: ответ маленький, а лишнее нам не нужно.
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return "", fmt.Errorf("ucaller: read response: %w", err)
	}

	// При ошибке uCaller отвечает {"status": false, "error": "...", "code": 3}.
	var out struct {
		Status bool   `json:"status"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("ucaller: status %d: decode response: %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK || !out.Status {
		return "", fmt.Errorf("ucaller: status %d: %s", resp.StatusCode, raw)
	}
	return code, nil
}
