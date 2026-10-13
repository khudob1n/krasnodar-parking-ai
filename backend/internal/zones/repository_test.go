package zones_test

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/db"
	"github.com/khudob1n/krasnodar-parking-ai/backend/internal/zones"
)

// Этот тест ходит в настоящую базу. Без TEST_DATABASE_URL он пропускается.
func TestRepositoryUpsertAndGet(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()

	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE zones CASCADE"); err != nil {
		t.Fatal(err)
	}

	repo := zones.NewRepository(pool)
	zone := zones.Zone{
		Number:   1103,
		Kind:     zones.KindStreet,
		Address:  "ул. Комсомольская",
		Location: zones.Point{Lat: 45.019142, Lon: 38.969612},
		Segments: []zones.Segment{
			{Start: zones.Point{Lat: 45.019282, Lon: 38.96891}, Stop: zones.Point{Lat: 45.019203, Lon: 38.969379}},
			{Start: zones.Point{Lat: 45.019178, Lon: 38.969532}, Stop: zones.Point{Lat: 45.01915, Lon: 38.96971}},
		},
		PricePerHour: 6000,
		PaidFrom:     "08:00",
		PaidTo:       "19:00",
		TotalPlaces:  17,
		TariffText:   "60.0 руб/час",
	}
	free := zones.Zone{
		Number: 12001, Kind: zones.KindFree, Address: "ул. Московская",
		Location: zones.Point{Lat: 45.05096, Lon: 39.003017},
		Segments: []zones.Segment{}, PaidFrom: "00:00", PaidTo: "00:00",
	}

	if err := repo.Upsert(ctx, []zones.Zone{zone, free}); err != nil {
		t.Fatal(err)
	}
	// Повторный импорт не должен падать и не должен плодить дубли.
	zone.PricePerHour = 4000
	if err := repo.Upsert(ctx, []zones.Zone{zone}); err != nil {
		t.Fatal(err)
	}

	for _, want := range []zones.Zone{zone, free} {
		got, err := repo.Get(ctx, want.Number)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("zone %d:\n got %+v\nwant %+v", want.Number, got, want)
		}
	}

	var n int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM zones").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("want 2 rows, got %d", n)
	}

	if _, err := repo.Get(ctx, 999999); !errors.Is(err, zones.ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

func TestRepositoryNearby(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()

	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE zones CASCADE"); err != nil {
		t.Fatal(err)
	}

	// Точка зоны 1103 стоит в начале улицы, а участок тянется на ~130 м
	// на восток. Мы стоим у дальнего конца участка.
	street := zones.Zone{
		Number: 1103, Kind: zones.KindStreet, Address: "ул. Комсомольская",
		Location: zones.Point{Lat: 45.019282, Lon: 38.96891},
		Segments: []zones.Segment{{
			Start: zones.Point{Lat: 45.019282, Lon: 38.96891},
			Stop:  zones.Point{Lat: 45.019004, Lon: 38.970557},
		}},
		PaidFrom: "08:00", PaidTo: "19:00",
	}
	lot := zones.Zone{
		Number: 1158, Kind: zones.KindLot, Address: "Привокзальная площадь",
		Location: zones.Point{Lat: 45.019072, Lon: 38.988583},
		PaidFrom: "00:00", PaidTo: "00:00",
	}
	repo := zones.NewRepository(pool)
	if err := repo.Upsert(ctx, []zones.Zone{street, lot}); err != nil {
		t.Fatal(err)
	}
	me := zones.Point{Lat: 45.019004, Lon: 38.970557}

	got, err := repo.Nearby(ctx, zones.NearbyQuery{Point: me, RadiusMeters: 500, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Number != 1103 {
		t.Fatalf("radius 500: want only zone 1103, got %+v", got)
	}
	if got[0].DistanceMeters != 0 {
		t.Errorf("distance must be measured to the segment, got %d m", got[0].DistanceMeters)
	}

	got, err = repo.Nearby(ctx, zones.NearbyQuery{Point: me, RadiusMeters: 2000, Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Number != 1103 || got[1].Number != 1158 {
		t.Fatalf("radius 2000: want [1103 1158], got %+v", got)
	}
	if d := got[1].DistanceMeters; d < 1300 || d > 1500 {
		t.Errorf("distance to 1158: want ~1400 m, got %d", d)
	}

	got, err = repo.Nearby(ctx, zones.NearbyQuery{Point: me, RadiusMeters: 2000, Limit: 10, Kinds: []zones.Kind{zones.KindLot}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Number != 1158 {
		t.Fatalf("kind=lot: want only 1158, got %+v", got)
	}
}

func TestImportFileIfEmpty(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()

	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE zones CASCADE"); err != nil {
		t.Fatal(err)
	}
	repo := zones.NewRepository(pool)

	n, err := repo.ImportFileIfEmpty(ctx, "../../../data/zones.json")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1716 {
		t.Fatalf("imported %d zones, want 1716", n)
	}
	// Второй запуск (перезапуск сервера) ничего не трогает.
	if n, err = repo.ImportFileIfEmpty(ctx, "../../../data/zones.json"); err != nil || n != 0 {
		t.Fatalf("second import: n=%d err=%v, want 0, nil", n, err)
	}
	if count, err := repo.Count(ctx); err != nil || count != 1716 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}
