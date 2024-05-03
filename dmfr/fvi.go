package dmfr

import (
	"strconv"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
)

// FeedVersionImport .
type FeedVersionImport struct {
	ImportLog                 string
	ExceptionLog              string
	ImportLevel               int  // deprecated
	Success                   bool // Finished, Success Yes/No
	InProgress                bool // In Progress
	ScheduleRemoved           bool // Stop times and trips have been uimported
	InterpolatedStopTimeCount int
	EntityCount               tt.Counts
	WarningCount              tt.Counts
	GeneratedCount            tt.Counts
	SkipEntityErrorCount      tt.Counts
	SkipEntityReferenceCount  tt.Counts
	SkipEntityFilterCount     tt.Counts
	SkipEntityMarkedCount     tt.Counts
	tl.DatabaseEntity
	tl.FeedVersionEntity
	tl.Timestamps
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

// GetID returns the ID
func (fvi *FeedVersionImport) GetID() int {
	return fvi.ID
}

// SetID sets the ID.
func (fvi *FeedVersionImport) SetID(v int) {
	fvi.ID = v
}

// EntityID .
func (fvi *FeedVersionImport) EntityID() string {
	return strconv.Itoa(fvi.ID)
}

// TableName .
func (FeedVersionImport) TableName() string {
	return "feed_version_gtfs_imports"
}
