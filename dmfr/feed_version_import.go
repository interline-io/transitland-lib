package dmfr

import (
	"strconv"

	"github.com/interline-io/transitland-lib/tt"
)

// Import sources, recorded on FeedVersionImport.ImportSource, indicate whether
// an import was initiated by a user or by an automatic/maintenance process.
// Garbage collection uses this to retain user-initiated imports longer.
const (
	ImportSourceAutomatic = "automatic" // queued or run by a maintenance process
	ImportSourceManual    = "manual"    // initiated by a user
)

// FeedVersionImport is a record of when a feed version was imported into the database.
type FeedVersionImport struct {
	ImportLog                 string
	ExceptionLog              string
	ImportLevel               int    // deprecated
	ImportSource              string // "automatic" or "manual"; see ImportSource* constants
	Success                   bool   // Finished, Success Yes/No
	InProgress                bool   // In Progress
	ScheduleRemoved           bool   // Stop times and trips have been uimported
	InterpolatedStopTimeCount int
	EntityCount               tt.Counts
	WarningCount              tt.Counts
	GeneratedCount            tt.Counts
	SkipEntityErrorCount      tt.Counts
	SkipEntityReferenceCount  tt.Counts
	SkipEntityFilterCount     tt.Counts
	SkipEntityMarkedCount     tt.Counts
	tt.DatabaseEntity
	tt.FeedVersionEntity
	tt.Timestamps
}

// NewFeedVersionImport returns an initialized FeedVersionImport.
func NewFeedVersionImport() *FeedVersionImport {
	fvi := FeedVersionImport{}
	fvi.EntityCount = tt.Counts{}
	fvi.WarningCount = tt.Counts{}
	fvi.GeneratedCount = tt.Counts{}
	fvi.SkipEntityErrorCount = tt.Counts{}
	fvi.SkipEntityReferenceCount = tt.Counts{}
	fvi.SkipEntityFilterCount = tt.Counts{}
	fvi.SkipEntityMarkedCount = tt.Counts{}
	return &fvi
}

// EntityID .
func (fvi *FeedVersionImport) EntityID() string {
	return strconv.Itoa(fvi.ID)
}

// TableName .
func (FeedVersionImport) TableName() string {
	return "feed_version_gtfs_imports"
}
