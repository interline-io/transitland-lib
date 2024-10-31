package dmfr

import (
	"strconv"

	"github.com/interline-io/transitland-lib/tt"
)

// FeedVersionServiceLevel is a cached summary of the number of seconds of scheduled service for a GTFS feed on a week-by-week basis.
type FeedVersionServiceLevel struct {
	StartDate tt.Date
	EndDate   tt.Date
	Monday    int
	Tuesday   int
	Wednesday int
	Thursday  int
	Friday    int
	Saturday  int
	Sunday    int
	tt.FeedVersionEntity
	tt.DatabaseEntity
}

// EntityID .
func (fvi *FeedVersionServiceLevel) EntityID() string {
	return strconv.Itoa(fvi.ID)
}

// TableName .
func (FeedVersionServiceLevel) TableName() string {
	return "feed_version_service_levels"
}

func (fvsl *FeedVersionServiceLevel) Total() int {
	return fvsl.Monday + fvsl.Tuesday + fvsl.Wednesday + fvsl.Thursday + fvsl.Friday + fvsl.Saturday + fvsl.Sunday
}
