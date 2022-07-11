package tl

import (
	"time"

	tt "github.com/interline-io/transitland-lib/tl/tt"
	geom "github.com/twpayne/go-geom"
)

type String = tt.String
type Strings = tt.Strings
type Int = tt.Int
type Float = tt.Float
type Key = tt.Key
type Time = tt.Time
type Date = tt.Date
type Tags = tt.Tags
type Ints = tt.Ints
type IntEnum = tt.IntEnum
type WideTime = tt.WideTime
type Currency = tt.Currency
type Language = tt.Language
type Timezone = tt.Timezone
type Point = tt.Point
type Polygon = tt.Polygon
type Geometry = tt.Geometry[geom.T]
type LineString = tt.LineString
type Color = tt.Color
type Url = tt.Url
type Email = tt.Email

func NewString(v string) String {
	return tt.NewString(v)
}

func NewInt(v int) Int {
	return tt.NewInt(v)
}

func NewFloat(v float64) Float {
	return tt.NewFloat(v)
}

func NewKey(v string) Key {
	return tt.NewKey(v)
}

func NewTime(t time.Time) Time {
	return tt.NewTime(t)
}

func NewDate(t time.Time) Date {
	return tt.NewDate(t)
}

func NewInts(v []int) Ints {
	return tt.NewInts(v)
}

func NewTimezone(v string) Timezone {
	return tt.NewTimezone(v)
}

func NewWideTimeFromSeconds(v int) WideTime {
	return tt.NewWideTimeFromSeconds(v)
}

func NewWideTime(v string) (WideTime, error) {
	return tt.NewWideTime(v)
}

func NewCurrency(v string) Currency {
	return tt.NewCurrency(v)
}

func NewIntEnum(v int) IntEnum {
	return tt.NewIntEnum(v)
}

func NewPoint(lon, lat float64) Point {
	return tt.NewPoint(lon, lat)
}
