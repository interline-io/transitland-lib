package gotransit

// Reader defines an interface for reading entities from a GTFS feed.
type Reader interface {
	Open() error
	Close() error
	ValidateStructure() []error
	StopTimesByTripID(...string) chan []StopTime
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
	Trips() chan Trip
}
