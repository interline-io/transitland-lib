package tl

import (
	"time"

	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/enum"
)

type String = enum.String
type Strings = enum.Strings
type Int = enum.Int
type Float = enum.Float
type Key = enum.Key
type Time = enum.Time
type Date = enum.Date
type Tags = enum.Tags
type IntSlice = enum.IntSlice
type IntEnum = enum.IntEnum
type WideTime = enum.WideTime
type Currency = enum.Currency
type Language = enum.Language
type Timezone = enum.Timezone
type Point = enum.Point
type Polygon = enum.Polygon
type Geometry = enum.Geometry
type LineString = enum.LineString

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

func NewIntSlice(v []int) IntSlice {
	return enum.NewIntSlice(v)
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

/////////

func CheckError(a []error, v error) []error {
	if v != nil {
		a = append(a, v)
	}
	return a
}

func CheckFieldError(field string, v error) error {
	if c, ok := v.(causes.CanUpdate); ok {
		c.Update(&causes.Context{Field: field})
	}
	return v
}

func CheckValidPresent(field string, value isEnum) error {
	err := value.Error()
	if err != nil {
		//
	} else if value.String() == "" {
		err = causes.NewRequiredFieldError(field)
	} else if !value.IsValid() {
		err = causes.NewInvalidFieldError(field, value.String(), nil)
	}
	if c, ok := err.(causes.CanUpdate); ok {
		c.Update(&causes.Context{Field: field})
	}
	return err
}

type isEnum interface {
	String() string
	Error() error
	IsValid() bool
}
