package dmfr

import (
	"strconv"
	"time"

	"github.com/interline-io/transitland-lib/tt"
)

// FeedVersion represents a single version of a GTFS data source.
type FeedVersion struct {
	FeedID               int
	SHA1                 string
	SHA1Dir              string
	File                 string
	URL                  string
	EarliestCalendarDate tt.Date
	LatestCalendarDate   tt.Date
	FetchedAt            time.Time
	Fragment             tt.String
	Name                 tt.String
	Description          tt.String
	CreatedBy            tt.String
	UpdatedBy            tt.String
	tt.DatabaseEntity
	tt.Timestamps
}

// EntityID .
func (ent *FeedVersion) EntityID() string {
	return strconv.Itoa(ent.ID)
}

// TableName sets the table name prefix.
func (ent *FeedVersion) TableName() string {
	return "feed_versions"
}
