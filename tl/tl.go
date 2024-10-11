// Package tl provides the core types and utility functions for transitland-lib.
package tl

import (
	_ "embed"
	"runtime/debug"
	"strings"

	adapters "github.com/interline-io/transitland-lib/adapters"
	dmfr2 "github.com/interline-io/transitland-lib/dmfr2"
	gtfs "github.com/interline-io/transitland-lib/gtfs"
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

////////

type FeedVersion = dmfr2.FeedVersion
type Operator = dmfr2.Operator
type OperatorAssociatedFeed = dmfr2.OperatorAssociatedFeed
type OperatorAssociatedFeeds = dmfr2.OperatorAssociatedFeeds

type Reader = adapters.Reader
type Writer = adapters.Writer
type WriterWithExtraColumns = adapters.WriterWithExtraColumns

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
type Transfer = gtfs.Transfer
type Translation = gtfs.Translation
type Trip = gtfs.Trip
type StopTime = gtfs.StopTime
type FareRule = gtfs.FareRule

type Entity = tt.Entity
type EntityWithReferences = tt.EntityWithReferences
type EntityWithID = tt.EntityWithID
type EntityWithErrors = tt.EntityWithErrors
type EntityWithExtra = tt.EntityWithExtra

type MinEntity = tt.MinEntity
type LineEntity = tt.LineEntity
type ExtraEntity = tt.ExtraEntity
type ReferenceEntity = tt.ReferenceEntity
type FeedVersionEntity = tt.FeedVersionEntity
type DatabaseEntity = tt.DatabaseEntity
type Timestamps = tt.Timestamps
type BaseEntity = tt.BaseEntity

type EntityMap = tt.EntityMap

func NewEntityMap() *EntityMap {
	return tt.NewEntityMap()
}
