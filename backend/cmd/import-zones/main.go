// Команда import-zones загружает зоны из JSON-выгрузки parkingkrd.ru в базу.
//
//	go run ./cmd/import-zones -file ../data/zones.json
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/config"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/db"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/zones"
)

func main() {
	file := flag.String("file", "../data/zones.json", "путь к JSON с зонами")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(context.Background(), *file, log); err != nil {
		log.Error("import failed", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, path string, log *slog.Logger) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	list, err := zones.ParseSource(f)
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		return err
	}

	if err := zones.NewRepository(pool).Upsert(ctx, list); err != nil {
		return err
	}

	counts := map[zones.Kind]int{}
	for _, z := range list {
		counts[z.Kind]++
	}
	log.Info("zones imported", "total", len(list),
		"street", counts[zones.KindStreet], "lot", counts[zones.KindLot], "free", counts[zones.KindFree])
	return nil
}
