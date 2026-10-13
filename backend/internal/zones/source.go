package zones

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
	"time"
)

// sourceFile - корень JSON, снятого с parkingkrd.ru.
type sourceFile struct {
	Data []sourceZone `json:"data"`
}

// sourceZone повторяет поля сайта как есть, включая их названия.
type sourceZone struct {
	ID                   int             `json:"id"`
	Address              string          `json:"address"`
	Latitude             float64         `json:"latitude"`
	Longitude            float64         `json:"longitude"`
	TimeFrom             string          `json:"timeFrom"`
	TimeTo               string          `json:"timeTo"`
	Cost                 float64         `json:"cost"`
	MaxPlaces            int             `json:"maxPlaces"`
	DisabledPersonPlaces int             `json:"disabledPersonPlaces"`
	Description          string          `json:"description"`
	Itudes               []sourceSegment `json:"itudes"`
	ParkingForm          int             `json:"parkingForm"`
	Tarif                string          `json:"tarif"`
	InBuilding           bool            `json:"inBuilding"`
}

type sourceSegment struct {
	StartLatitude  float64 `json:"startLatitude"`
	StartLongitude float64 `json:"startLongitude"`
	StopLatitude   float64 `json:"stopLatitude"`
	StopLongitude  float64 `json:"stopLongitude"`
}

// ParseSource читает выгрузку сайта и превращает её в наши зоны.
func ParseSource(r io.Reader) ([]Zone, error) {
	var f sourceFile
	if err := json.NewDecoder(r).Decode(&f); err != nil {
		return nil, fmt.Errorf("decode zones json: %w", err)
	}

	result := make([]Zone, 0, len(f.Data))
	seen := make(map[int]bool, len(f.Data))
	for _, s := range f.Data {
		z, err := s.toZone()
		if err != nil {
			return nil, fmt.Errorf("zone %d: %w", s.ID, err)
		}
		if seen[z.Number] {
			return nil, fmt.Errorf("zone %d: duplicate number", z.Number)
		}
		seen[z.Number] = true
		result = append(result, z)
	}
	return result, nil
}

func (s sourceZone) toZone() (Zone, error) {
	kind, err := kindFromForm(s.ParkingForm)
	if err != nil {
		return Zone{}, err
	}
	if err := checkClock(s.TimeFrom); err != nil {
		return Zone{}, fmt.Errorf("timeFrom: %w", err)
	}
	if err := checkClock(s.TimeTo); err != nil {
		return Zone{}, fmt.Errorf("timeTo: %w", err)
	}
	if s.Cost < 0 {
		return Zone{}, fmt.Errorf("negative cost %v", s.Cost)
	}

	segments := make([]Segment, 0, len(s.Itudes))
	for _, it := range s.Itudes {
		seg := Segment{
			Start: Point{Lat: it.StartLatitude, Lon: it.StartLongitude},
			Stop:  Point{Lat: it.StopLatitude, Lon: it.StopLongitude},
		}
		// У площадок сайт отдаёт «участок» из одной точки. Линией это
		// не является, поэтому такие участки отбрасываем.
		if seg.Start == seg.Stop {
			continue
		}
		segments = append(segments, seg)
	}

	return Zone{
		Number:         s.ID,
		Kind:           kind,
		Address:        strings.TrimSpace(s.Address),
		Description:    strings.TrimSpace(s.Description),
		Location:       Point{Lat: s.Latitude, Lon: s.Longitude},
		Segments:       segments,
		PricePerHour:   int64(math.Round(s.Cost * 100)),
		PaidFrom:       s.TimeFrom,
		PaidTo:         s.TimeTo,
		TotalPlaces:    s.MaxPlaces,
		DisabledPlaces: s.DisabledPersonPlaces,
		InBuilding:     s.InBuilding,
		TariffText:     s.Tarif,
	}, nil
}

func kindFromForm(form int) (Kind, error) {
	switch form {
	case 0:
		return KindLot, nil
	case 1:
		return KindStreet, nil
	case 2:
		return KindFree, nil
	default:
		return "", fmt.Errorf("unknown parkingForm %d", form)
	}
}

// checkClock проверяет, что строка - время вида "08:00".
func checkClock(s string) error {
	if _, err := time.Parse("15:04", s); err != nil {
		return fmt.Errorf("bad time %q", s)
	}
	return nil
}
