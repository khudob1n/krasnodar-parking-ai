package auth

import (
	"context"
	"log/slog"
)

// LogSender вместо отправки SMS пишет текст в лог. Только для разработки.
type LogSender struct{ Log *slog.Logger }

func (s LogSender) Send(_ context.Context, phone, text string) error {
	s.Log.Warn("SMS not sent, dev mode", "phone", phone, "text", text)
	return nil
}
