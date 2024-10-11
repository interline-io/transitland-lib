package tl

import tl "github.com/interline-io/transitland-lib/gtfs"

// Reader defines an interface for reading entities from a GTFS feed.
type Reader interface {
	Open() error
	Close() error
	ValidateStructure() []error
	StopTimesByTripID(...string) chan []tl.StopTime
	String() string
	// Entities
	ReadEntities(c interface{}) error
	Stops() chan tl.Stop
	StopTimes() chan tl.StopTime
	Agencies() chan tl.Agency
	Calendars() chan tl.Calendar
	CalendarDates() chan tl.CalendarDate
	FareAttributes() chan tl.FareAttribute
	FareRules() chan tl.FareRule
	FeedInfos() chan tl.FeedInfo
	Frequencies() chan tl.Frequency
	Routes() chan tl.Route
	Shapes() chan tl.Shape
	Transfers() chan tl.Transfer
	Pathways() chan tl.Pathway
	Levels() chan tl.Level
	Trips() chan tl.Trip
	Translations() chan tl.Translation
	Attributions() chan tl.Attribution
	Areas() chan tl.Area
	StopAreas() chan tl.StopArea
	FareLegRules() chan tl.FareLegRule
	FareTransferRules() chan tl.FareTransferRule
	FareProducts() chan tl.FareProduct
	RiderCategories() chan tl.RiderCategory
	FareMedia() chan tl.FareMedia
}
