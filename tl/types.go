package tl

import (
	"time"

	"github.com/interline-io/transitland-lib/tl/enum"
	"github.com/twpayne/go-geom"
)

type String = enum.String
type Strings = enum.Strings
type Int = enum.Int
type Float = enum.Float
type Key = enum.Key
type Time = enum.Time
type Date = enum.Date
type Tags = enum.Tags
type Ints = enum.Ints
type IntEnum = enum.IntEnum
type WideTime = enum.WideTime
type Currency = enum.Currency
type Language = enum.Language
type Timezone = enum.Timezone
type Point = enum.Point
type Polygon = enum.Polygon
type Geometry = enum.Geometry[geom.T]
type LineString = enum.LineString
type Color = enum.Color
type Url = enum.Url
type Email = enum.Email

func NewString(v string) String {
	return enum.NewString(v)
}

func NewInt(v int) Int {
	return enum.NewInt(v)
}

func NewFloat(v float64) Float {
	return enum.NewFloat(v)
}

func NewKey(v string) Key {
	return enum.NewKey(v)
}

func NewTime(t time.Time) Time {
	return enum.NewTime(t)
}

func NewDate(t time.Time) Date {
	return enum.NewDate(t)
}

func NewInts(v []int) Ints {
	return enum.NewInts(v)
}

func NewTimezone(v string) Timezone {
	return enum.NewTimezone(v)
}

func NewWideTimeFromSeconds(v int) WideTime {
	return enum.NewWideTimeFromSeconds(v)
}

func NewWideTime(v string) (WideTime, error) {
	return enum.NewWideTime(v)
}

func NewCurrency(v string) Currency {
	return enum.NewCurrency(v)
}

func NewIntEnum(v int) IntEnum {
	return enum.NewIntEnum(v)
}

func NewPoint(lon, lat float64) Point {
	return enum.NewPoint(lon, lat)
}
