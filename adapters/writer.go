package adapters

import "github.com/interline-io/transitland-lib/tl/tt"

type Entity = tt.Entity

// Writer writes a GTFS feed.
type Writer interface {
	Open() error
	Close() error
	Create() error
	Delete() error
	NewReader() (Reader, error)
	AddEntity(Entity) (string, error)
	AddEntities([]Entity) ([]string, error)
	String() string
}

type WriterWithExtraColumns interface {
	Writer
	WriteExtraColumns(bool)
}
