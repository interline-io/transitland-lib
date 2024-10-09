// Package tl provides the core types and utility functions for transitland-lib.
package tl

import (
	_ "embed"
	"runtime/debug"
	"strings"

	"github.com/interline-io/transitland-lib/tl/tt"
)

// Read version from compiled in git details
var Version VersionInfo

func init() {
	Version = getVersion()
}

type VersionInfo struct {
	Tag        string
	Commit     string
	CommitTime string
}

func getVersion() VersionInfo {
	ret := VersionInfo{}
	info, _ := debug.ReadBuildInfo()
	tagPrefix := "main.tag="
	for _, kv := range info.Settings {
		switch kv.Key {
		case "vcs.revision":
			ret.Commit = kv.Value
		case "vcs.time":
			ret.CommitTime = kv.Value
		case "-ldflags":
			for _, ss := range strings.Split(kv.Value, " ") {
				if strings.HasPrefix(ss, tagPrefix) {
					ret.Tag = strings.TrimPrefix(ss, tagPrefix)
				}
			}
		}
	}
	return ret
}

// GTFSVERSION is the commit for the spec reference.md file.
var GTFSVERSION = "11a49075c1f50d0130b934833b7eeb3fe518961c"

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
