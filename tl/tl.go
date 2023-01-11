package tl

import (
	"github.com/interline-io/transitland-lib/tl/tt"
)

// VERSION is the current software version.
var VERSION = "v0.12.0"

// GTFSVERSION is the commit for the spec reference.md file.
var GTFSVERSION = "f62ca848efa843fdf1e6f9856aaff50627c3ed69"

// GTFSRTVERSION is the commit for the gtfs-realtime.proto file.
var GTFSRTVERSION = "6fcc3800b15954227af7335d571791738afb1a67"

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
