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

// Standardized sort directions. The empty string (default/omitted) means no sort.
const (
	SortAsc  = "asc"
	SortDesc = "desc"
)

type StandardizedSortOptions struct {
	StandardizedSort        string   // SortAsc, SortDesc, or "" (no sort).
	StandardizedSortColumns []string // Optional: specific columns to sort by. If empty, defaults are used.
}

// SortableWriter is the minimal interface for accepting a sort config.
type SortableWriter interface {
	SetStandardizedSortOptions(StandardizedSortOptions)
}

// WriterWithStandardizedSort is a full Writer that also accepts sort config.
type WriterWithStandardizedSort interface {
	Writer
	SortableWriter
}
