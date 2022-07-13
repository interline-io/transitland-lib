package tl

import (
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
