// Package zones хранит парковочные зоны Краснодара: модель, разбор выгрузки
// с parkingkrd.ru и работу с базой.
package zones

// Kind - тип парковки.
type Kind string

const (
	KindStreet Kind = "street" // платная парковка вдоль улицы (участки улиц)
	KindLot    Kind = "lot"    // платная площадка (одна точка на карте)
	KindFree   Kind = "free"   // бесплатная парковка
)

// Point - точка на карте в градусах WGS 84.
type Point struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// Segment - участок улицы, вдоль которого разрешено парковаться.
type Segment struct {
	Start Point `json:"start"`
	Stop  Point `json:"stop"`
}

// Zone - парковочная зона. Number - номер с таблички, он же ключ.
type Zone struct {
	Number      int       `json:"number"`
	Kind        Kind      `json:"kind"`
	Address     string    `json:"address"`
	Description string    `json:"description,omitempty"`
	Location    Point     `json:"location"`
	Segments    []Segment `json:"segments"`

	// Цена часа в копейках в платные часы. Деньги храним целым числом,
	// чтобы не ловить ошибки округления float.
	PricePerHour int64 `json:"pricePerHour"`
	// Платные часы в формате "ЧЧ:ММ". Одинаковые значения ("00:00"-"00:00")
	// означают, что тариф действует круглосуточно.
	PaidFrom string `json:"paidFrom"`
	PaidTo   string `json:"paidTo"`

	TotalPlaces    int    `json:"totalPlaces"`
	DisabledPlaces int    `json:"disabledPlaces"`
	InBuilding     bool   `json:"inBuilding"`
	TariffText     string `json:"tariffText"`
}
