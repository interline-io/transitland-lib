package adapters

import gtfs "github.com/interline-io/transitland-lib/gtfs"

// Reader defines an interface for reading entities from a GTFS feed.
type Reader interface {
	Open() error
	Close() error
	ValidateStructure() []error
	StopTimesByTripID(...string) chan []gtfs.StopTime
	String() string
	// Entities
	ReadEntities(c interface{}) error
	Stops() chan gtfs.Stop
	StopTimes() chan gtfs.StopTime
	Agencies() chan gtfs.Agency
	Calendars() chan gtfs.Calendar
	CalendarDates() chan gtfs.CalendarDate
	FareAttributes() chan gtfs.FareAttribute
	FareRules() chan gtfs.FareRule
	FeedInfos() chan gtfs.FeedInfo
	Frequencies() chan gtfs.Frequency
	Routes() chan gtfs.Route
	Shapes() chan gtfs.Shape
	Transfers() chan gtfs.Transfer
	Pathways() chan gtfs.Pathway
	Levels() chan gtfs.Level
	Trips() chan gtfs.Trip
	Translations() chan gtfs.Translation
	Attributions() chan gtfs.Attribution
	Areas() chan gtfs.Area
	StopAreas() chan gtfs.StopArea
	FareLegRules() chan gtfs.FareLegRule
	FareTransferRules() chan gtfs.FareTransferRule
	FareProducts() chan gtfs.FareProduct
	RiderCategories() chan gtfs.RiderCategory
	FareMedia() chan gtfs.FareMedia
}
