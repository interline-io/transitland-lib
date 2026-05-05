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

// Standardized sort directions. The empty string means no sort.
const (
	SortAsc  = "asc"
	SortDesc = "desc"
	SortNone = "none"
)

type SortOptions struct {
	StandardizedSort        string   // SortAsc, SortDesc, or "" (no sort).
	StandardizedSortColumns []string // Optional: specific columns to sort by. If empty, defaults are used.
}

type StandardizedSortOptions = SortOptions

type WriterWithStandardizedSort interface {
	Writer
	SetStandardizedSortOptions(StandardizedSortOptions)
}
