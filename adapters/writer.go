package adapters

import "github.com/interline-io/transitland-lib/tt"

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

type SortOptions struct {
	StandardizedSort        string   // "asc" or "desc"
	StandardizedSortColumns []string // Optional: specific columns to sort by. If empty, defaults are used.
}

type StandardizedSortOptions = SortOptions

type WriterWithStandardizedSort interface {
	Writer
	SetStandardizedSortOptions(StandardizedSortOptions)
}
