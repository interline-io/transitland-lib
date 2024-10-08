package tlxy

import (
	"strconv"
	"strings"
)

type BoundingBox struct {
	MinLon float64 `json:"min_lon"`
	MinLat float64 `json:"min_lat"`
	MaxLon float64 `json:"max_lon"`
	MaxLat float64 `json:"max_lat"`
}

func (v *BoundingBox) Contains(pt Point) bool {
	if pt.Lon >= v.MinLon && pt.Lon <= v.MaxLon && pt.Lat >= v.MinLat && pt.Lat <= v.MaxLat {
		return true
	}
	return false
}

func ParseBbox(v string) (BoundingBox, error) {
	r := BoundingBox{}
	if s := strings.Split(v, ","); len(s) == 4 {
		r.MinLon, _ = strconv.ParseFloat(s[0], 64)
		r.MinLat, _ = strconv.ParseFloat(s[1], 64)
		r.MaxLon, _ = strconv.ParseFloat(s[2], 64)
		r.MaxLat, _ = strconv.ParseFloat(s[3], 64)
	}
	return r, nil
}
