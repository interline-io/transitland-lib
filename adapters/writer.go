package adapters

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tt"
)

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

// ValidateSortDirection returns nil if s is empty, SortAsc, or SortDesc;
// otherwise it returns an error listing the valid values. Empty is treated
// as valid because callers conventionally use it to mean "no sort".
func ValidateSortDirection(s string) error {
	switch s {
	case "", SortAsc, SortDesc:
		return nil
	}
	return fmt.Errorf("invalid sort direction %q (must be %q or %q)", s, SortAsc, SortDesc)
}

type StandardizedSortOptions struct {
	ApplySort   string   // SortAsc, SortDesc, or "" (no sort).
	SortColumns []string // Optional: specific columns to sort by. If empty, defaults are used.
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
