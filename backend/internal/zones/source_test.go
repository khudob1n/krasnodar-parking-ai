package zones

import (
	"os"
	"strings"
	"testing"
)

const sampleJSON = `{"data":[
 {"id":1103,"address":" ул. Комсомольская ","latitude":45.019142,"longitude":38.969612,
  "timeFrom":"08:00","timeTo":"19:00","cost":60.0,"maxPlaces":17,"disabledPersonPlaces":2,
  "description":"","parkingForm":1,"tarif":"60.0 руб/час","inBuilding":false,
  "itudes":[{"startLatitude":45.019282,"startLongitude":38.96891,"stopLatitude":45.019203,"stopLongitude":38.969379}]},
 {"id":1158,"address":"Привокзальная площадь","latitude":45.019072,"longitude":38.988583,
  "timeFrom":"00:00","timeTo":"00:00","cost":100.0,"maxPlaces":123,"disabledPersonPlaces":13,
  "description":"","parkingForm":0,"tarif":"100.0 руб/час круглосуточно","inBuilding":false,
  "itudes":[{"startLatitude":45.019072,"startLongitude":38.988583,"stopLatitude":45.019072,"stopLongitude":38.988583}]}
]}`

func TestParseSource(t *testing.T) {
	got, err := ParseSource(strings.NewReader(sampleJSON))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 zones, got %d", len(got))
	}

	street := got[0]
	if street.Number != 1103 || street.Kind != KindStreet {
		t.Errorf("street zone: got number %d kind %q", street.Number, street.Kind)
	}
	if street.Address != "ул. Комсомольская" {
		t.Errorf("address not trimmed: %q", street.Address)
	}
	if street.PricePerHour != 6000 {
		t.Errorf("price: want 6000 kopecks, got %d", street.PricePerHour)
	}
	if len(street.Segments) != 1 {
		t.Errorf("want 1 segment, got %d", len(street.Segments))
	}

	lot := got[1]
	if lot.Kind != KindLot {
		t.Errorf("lot zone kind: got %q", lot.Kind)
	}
	if len(lot.Segments) != 0 {
		t.Errorf("single-point segment must be dropped, got %d", len(lot.Segments))
	}
}

func TestParseSourceErrors(t *testing.T) {
	cases := map[string]string{
		"unknown form": `{"data":[{"id":1,"timeFrom":"00:00","timeTo":"00:00","parkingForm":7}]}`,
		"bad time":     `{"data":[{"id":1,"timeFrom":"8 утра","timeTo":"00:00","parkingForm":1}]}`,
		"duplicate": `{"data":[{"id":1,"timeFrom":"00:00","timeTo":"00:00","parkingForm":1},
		                       {"id":1,"timeFrom":"00:00","timeTo":"00:00","parkingForm":1}]}`,
		"not json": `[`,
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseSource(strings.NewReader(input)); err == nil {
				t.Error("want error, got nil")
			}
		})
	}
}

// Проверяем, что настоящая выгрузка из data/ разбирается целиком.
func TestParseRealFile(t *testing.T) {
	f, err := os.Open("../../../data/zones.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	got, err := ParseSource(f)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[Kind]int{}
	for _, z := range got {
		counts[z.Kind]++
	}
	want := map[Kind]int{KindStreet: 304, KindLot: 19, KindFree: 1393}
	for k, n := range want {
		if counts[k] != n {
			t.Errorf("%s: want %d, got %d", k, n, counts[k])
		}
	}
}
