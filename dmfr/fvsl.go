package dmfr

import (
	"strconv"

	"github.com/interline-io/transitland-lib/tl"
)

// FeedVersionServiceLevel .
type FeedVersionServiceLevel struct {
	StartDate tl.Date
	EndDate   tl.Date
	Monday    int
	Tuesday   int
	Wednesday int
	Thursday  int
	Friday    int
	Saturday  int
	Sunday    int
	tl.FeedVersionEntity
	tl.DatabaseEntity
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
