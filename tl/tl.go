// Package tl provides the core types and utility functions for transitland-lib.
package tl

import (
	_ "embed"
	"runtime/debug"
	"strings"

	"github.com/interline-io/transitland-lib/tl/gtfs"
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

type Agency = gtfs.Agency
type Area = gtfs.Area
type Attribution = gtfs.Attribution
type Calendar = gtfs.Calendar
type CalendarDate = gtfs.CalendarDate
type FareAttribute = gtfs.FareAttribute
type FareLegRule = gtfs.FareLegRule
type FareMedia = gtfs.FareMedia
type FareProduct = gtfs.FareProduct
type FareTransferRule = gtfs.FareTransferRule
type FeedInfo = gtfs.FeedInfo
type Frequency = gtfs.Frequency
type Level = gtfs.Level
type Pathway = gtfs.Pathway
type RiderCategory = gtfs.RiderCategory
type Route = gtfs.Route
type Shape = gtfs.Shape
type Stop = gtfs.Stop
type StopArea = gtfs.StopArea
type StopTime = gtfs.StopTime
type Trip = gtfs.Trip
type FareRule = gtfs.FareRule
type Translation = gtfs.Translation
type Transfer = gtfs.Transfer

type Entity = gtfs.Entity
type EntityWithErrors = gtfs.EntityWithErrors
type DatabaseEntity = gtfs.DatabaseEntity
type Timestamps = gtfs.Timestamps
type FeedVersionEntity = gtfs.FeedVersionEntity
type MinEntity = gtfs.MinEntity
type BaseEntity = gtfs.BaseEntity
type EntityWithReferences = gtfs.EntityWithReferences
type EntityWithExtra = gtfs.EntityWithExtra
type EntityWithID = gtfs.EntityWithID

type EntityMap = gtfs.EntityMap

func NewEntityMap() *EntityMap {
	return gtfs.NewEntityMap()
}

func NewShapeFromShapes(shapes []Shape) Shape {
	return gtfs.NewShapeFromShapes(shapes)
}

func ValidateStopTimes(stoptimes []StopTime) []error {
	return gtfs.ValidateStopTimes(stoptimes)
}
