package httpapi

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// allZonesTTL - сколько держать готовый ответ в памяти. Зоны меняются только
// при импорте, так что раз в несколько минут сходить в базу достаточно.
const allZonesTTL = 5 * time.Minute

// allZones отдаёт все зоны одним ответом. Ответ ~1 МБ JSON, поэтому он сжат
// gzip (~150 КБ) и помечен ETag: приложение присылает If-None-Match и, если
// зоны не менялись, получает пустой 304 вместо мегабайта.
type allZones struct {
	store ZoneStore
	log   *slog.Logger

	mu      sync.Mutex
	body    []byte // gzip
	etag    string
	expires time.Time
}

func (a *allZones) load(r *http.Request) (body []byte, etag string, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.body != nil && now().Before(a.expires) {
		return a.body, a.etag, nil
	}
	list, err := a.store.All(r.Context())
	if err != nil {
		return nil, "", err
	}
	raw, err := json.Marshal(map[string]any{"zones": list})
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(raw)
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(raw); err != nil {
		return nil, "", err
	}
	if err := zw.Close(); err != nil {
		return nil, "", err
	}
	a.body, a.etag, a.expires = buf.Bytes(), `"`+hex.EncodeToString(sum[:8])+`"`, now().Add(allZonesTTL)
	return a.body, a.etag, nil
}

func (a *allZones) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, etag, err := a.load(r)
	if err != nil {
		a.log.Error("all zones", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Vary", "Accept-Encoding")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		// Клиент без gzip (curl без --compressed) - распаковать для него.
		zr, err := gzip.NewReader(bytes.NewReader(body))
		if err == nil {
			defer zr.Close()
			w.WriteHeader(http.StatusOK)
			if _, err := io.Copy(w, zr); err != nil {
				a.log.Warn("all zones: write", "err", err)
			}
			return
		}
	}
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Content-Length", fmt.Sprint(len(body)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}
