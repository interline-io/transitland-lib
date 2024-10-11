package adapters

import "github.com/interline-io/transitland-lib/tl/tt"

// Writer writes a GTFS feed.
type Writer interface {
	Open() error
	Close() error
	Create() error
	Delete() error
	NewReader() (Reader, error)
	AddEntity(tt.Entity) (string, error)
	AddEntities([]tt.Entity) ([]string, error)
	String() string
}

type WriterWithExtraColumns interface {
	Writer
	WriteExtraColumns(bool)
}
