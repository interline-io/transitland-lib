package dmfr

import (
	"strconv"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
)

type FeedVersionServiceWindow struct {
	FeedStartDate        tt.Date
	FeedEndDate          tt.Date
	EarliestCalendarDate tt.Date
	LatestCalendarDate   tt.Date
	FallbackWeek         tt.Date
	DefaultTimezone      tt.String
	tl.FeedVersionEntity
	tl.DatabaseEntity
	tl.Timestamps
}

func (fvi *FeedVersionServiceWindow) EntityID() string {
	return strconv.Itoa(fvi.ID)
}

func (FeedVersionServiceWindow) TableName() string {
	return "feed_version_service_windows"
}
