package auth

import (
	"context"
	"log/slog"
)

// LogSender никуда не звонит: придумывает код и пишет его в лог.
// Только для разработки.
type LogSender struct{ Log *slog.Logger }

func (s LogSender) SendCode(_ context.Context, phone string) (string, error) {
	code, err := randomCode()
	if err != nil {
		return "", err
	}
	s.Log.Warn("login code not sent, dev mode", "phone", phone, "code", code)
	return code, nil
}
