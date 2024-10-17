package adapters

import "github.com/interline-io/transitland-lib/gtfs"

// Reader defines an interface for reading entities from a GTFS feed.
type Reader interface {
	Open() error
	Close() error
	ValidateStructure() []error
	String() string
	// Entities
	ReadEntities(c interface{}) error
	gtfs.Reader
}
