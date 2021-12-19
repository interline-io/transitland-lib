package dmfr

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/interline-io/transitland-lib/tl"
)

// FeedVersionImport .
type FeedVersionImport struct {
	ID                        int
	FeedVersionID             int
	ImportLog                 string
	ExceptionLog              string
	ImportLevel               int  // deprecated
	Success                   bool // Finished, Success Yes/No
	InProgress                bool // In Progress
	ScheduleRemoved           bool // Stop times and trips have been uimported
	InterpolatedStopTimeCount int
	EntityCount               EntityCounter
	WarningCount              EntityCounter
	GeneratedCount            EntityCounter
	SkipEntityErrorCount      EntityCounter
	SkipEntityReferenceCount  EntityCounter
	SkipEntityFilterCount     EntityCounter
	SkipEntityMarkedCount     EntityCounter
	tl.Timestamps
}

// NewFeedVersionImport returns an initialized FeedVersionImport.
func NewFeedVersionImport() *FeedVersionImport {
	fvi := FeedVersionImport{}
	fvi.EntityCount = EntityCounter{}
	fvi.WarningCount = EntityCounter{}
	fvi.GeneratedCount = EntityCounter{}
	fvi.SkipEntityErrorCount = EntityCounter{}
	fvi.SkipEntityReferenceCount = EntityCounter{}
	fvi.SkipEntityFilterCount = EntityCounter{}
	fvi.SkipEntityMarkedCount = EntityCounter{}
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

// EntityCounter .
type EntityCounter map[string]int

// Value .
func (a EntityCounter) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan .
func (a *EntityCounter) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}
