package tl

// Reader defines an interface for reading entities from a GTFS feed.
type Reader interface {
	Open() error
	Close() error
	ValidateStructure() []error
	StopTimesByTripID(...string) chan []StopTime
	String() string
	// Entities
	ReadEntities(c interface{}) error
	Stops() chan Stop
	StopTimes() chan StopTime
	Agencies() chan Agency
	Calendars() chan Calendar
	CalendarDates() chan CalendarDate
	FareAttributes() chan FareAttribute
	FareRules() chan FareRule
	FeedInfos() chan FeedInfo
	Frequencies() chan Frequency
	Routes() chan Route
	Shapes() chan Shape
	Transfers() chan Transfer
	Pathways() chan Pathway
	Levels() chan Level
	Trips() chan Trip
	Translations() chan Translation
	Attributions() chan Attribution
	Areas() chan Area
	StopAreas() chan StopArea
	FareLegRules() chan FareLegRule
	FareTransferRules() chan FareTransferRule
	FareProducts() chan FareProduct
	RiderCategories() chan RiderCategory
	FareContainers() chan FareContainer
}
