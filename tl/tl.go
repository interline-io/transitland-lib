package tl

import (
	"time"

	"github.com/interline-io/transitland-lib/tl/tt"
)

// VERSION is the current software version.
var VERSION = "v0.10.3"

// GTFSVERSION is the commit for the spec reference.md file.
var GTFSVERSION = "2e6887ea16b689d8ebc70ad334ac8abb2f94a66e"

// GTFSRTVERSION is the commit for the gtfs-realtime.proto file.
var GTFSRTVERSION = "99fdfba627451d2861346c631af84535fa1e02fb"

// Aliases
type Date = tt.Date
type Float = tt.Float
type Geometry = tt.Geometry
type Int = tt.Int
type Ints = tt.Ints
type Key = tt.Key
type LineString = tt.LineString
type Point = tt.Point
type Polygon = tt.Polygon
type String = tt.String
type Strings = tt.Strings
type Tags = tt.Tags
type Time = tt.Time
type WideTime = tt.WideTime

func NewDate(v time.Time) Date                           { return tt.NewDate(v) }
func NewFloat(v float64) Float                           { return tt.NewFloat(v) }
func NewInt(v int) Int                                   { return tt.NewInt(v) }
func NewInts(v []int) Ints                               { return tt.NewInts(v) }
func NewKey(v string) Key                                { return tt.NewKey(v) }
func NewLineStringFromFlatCoords(v []float64) LineString { return tt.NewLineStringFromFlatCoords(v) }
func NewPoint(lon, lat float64) Point                    { return tt.NewPoint(lon, lat) }
func NewString(v string) String                          { return tt.NewString(v) }
func NewTime(v time.Time) Time                           { return tt.NewTime(v) }
func NewWideTime(v string) (WideTime, error)             { return tt.NewWideTime(v) }
func NewWideTimeFromSeconds(v int) WideTime              { return tt.NewWideTimeFromSeconds(v) }
func StringToSeconds(v string) (int, error)              { return tt.StringToSeconds(v) }
