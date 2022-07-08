package tl

import (
	"time"

	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tltypes"
)

type String = tltypes.String
type Strings = tltypes.Strings
type Int = tltypes.Int
type Float = tltypes.Float
type Key = tltypes.Key
type Time = tltypes.Time
type Date = tltypes.Date
type Tags = tltypes.Tags
type IntSlice = tltypes.IntSlice
type IntEnum = tltypes.IntEnum
type WideTime = tltypes.WideTime
type Currency = tltypes.Currency
type Language = tltypes.Language
type Timezone = tltypes.Timezone
type Point = tltypes.Point
type Polygon = tltypes.Polygon
type Geometry = tltypes.Geometry
type LineString = tltypes.LineString

func NewString(v string) String {
	return tltypes.NewString(v)
}

func NewInt(v int) Int {
	return tltypes.NewInt(v)
}

func NewFloat(v float64) Float {
	return tltypes.NewFloat(v)
}

func NewKey(v string) Key {
	return tltypes.NewKey(v)
}

func NewTime(t time.Time) Time {
	return tltypes.NewTime(t)
}

func NewDate(t time.Time) Date {
	return tltypes.NewDate(t)
}

func NewIntSlice(v []int) IntSlice {
	return tltypes.NewIntSlice(v)
}

func NewTimezone(v string) Timezone {
	return tltypes.NewTimezone(v)
}

func NewWideTimeFromSeconds(v int) WideTime {
	return tltypes.NewWideTimeFromSeconds(v)
}

func NewWideTime(v string) (WideTime, error) {
	return tltypes.NewWideTime(v)
}

func NewCurrency(v string) Currency {
	return tltypes.NewCurrency(v)
}

func NewIntEnum(v int) IntEnum {
	return tltypes.NewIntEnum(v)
}

func NewPoint(lon, lat float64) Point {
	return tltypes.NewPoint(lon, lat)
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

type isEnum interface {
	String() string
	Error() error
	IsValid() bool
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
