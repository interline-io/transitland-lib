package tl

import (
	"time"

	"github.com/interline-io/transitland-lib/tl/tt"
)

type String = tt.String
type Strings = tt.Strings
type Int = tt.Int
type Ints = tt.Ints
type Float = tt.Float
type Tags = tt.Tags
type Time = tt.Time
type Date = tt.Date
type Key = tt.Key
type WideTime = tt.WideTime
type LineString = tt.LineString
type Point = tt.Point
type Geometry = tt.Geometry
type Polygon = tt.Polygon

func NewString(v string) String                          { return tt.NewString(v) }
func NewFloat(v float64) Float                           { return tt.NewFloat(v) }
func NewKey(v string) Key                                { return tt.NewKey(v) }
func NewInt(v int) Int                                   { return tt.NewInt(v) }
func NewTime(v time.Time) Time                           { return tt.NewTime(v) }
func NewDate(v time.Time) Date                           { return tt.NewDate(v) }
func NewPoint(lon, lat float64) Point                    { return tt.NewPoint(lon, lat) }
func NewInts(v []int) Ints                               { return tt.NewInts(v) }
func NewWideTimeFromSeconds(v int) WideTime              { return tt.NewWideTimeFromSeconds(v) }
func NewWideTime(v string) (WideTime, error)             { return tt.NewWideTime(v) }
func StringToSeconds(v string) (int, error)              { return tt.StringToSeconds(v) }
func NewLineStringFromFlatCoords(v []float64) LineString { return tt.NewLineStringFromFlatCoords(v) }
