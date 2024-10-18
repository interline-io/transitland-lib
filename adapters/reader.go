package adapters

import "github.com/interline-io/transitland-lib/gtfs"

// Reader is the main interface for reading GTFS data
type Reader interface {
	EntityReader
	GtfsReader
}

// EntityReader defines methods for opening a reader, validating its structure, and reading entities through reflection
type EntityReader interface {
	Open() error
	Close() error
	ValidateStructure() []error
	String() string
	ReadEntities(c interface{}) error
}

// GtfsReader defines methods for accessing core GTFS entities
type GtfsReader interface {
	gtfs.Reader
	StopTimesByTripID(...string) chan []gtfs.StopTime
	ShapesByShapeID(...string) chan []gtfs.Shape
}
