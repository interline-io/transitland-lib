package dmfr

import (
	"strconv"

	"github.com/interline-io/transitland-lib/tt"
)

type FeedVersionServiceWindow struct {
	FeedStartDate        tt.Date
	FeedEndDate          tt.Date
	EarliestCalendarDate tt.Date
	LatestCalendarDate   tt.Date
	FallbackWeek         tt.Date
	DefaultTimezone      tt.String
	tt.FeedVersionEntity
	tt.DatabaseEntity
	tt.Timestamps
}

func (fvi *FeedVersionServiceWindow) EntityID() string {
	return strconv.Itoa(fvi.ID)
}

func (FeedVersionServiceWindow) TableName() string {
	return "feed_version_service_windows"
}
