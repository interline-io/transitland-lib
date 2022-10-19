package dmfr

import (
	"strconv"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
)

func NewFeedStatsFromReader(reader tl.Reader) (FeedVersionServiceWindow, []FeedVersionServiceLevel, error) {
	d := FeedVersionServiceWindow{}
	fvsls, err := NewFeedVersionServiceLevelsFromReader(reader)
	if err != nil {
		return d, nil, err
	}
	fvsw, err := NewFeedVersionServiceWindowFromReader(reader)
	if err != nil {
		return d, nil, err
	}
	dw, err := ServiceLevelDefaultWeek(fvsw.FeedStartDate, fvsw.FeedEndDate, fvsls)
	if err != nil {
		return d, nil, err
	}
	fvsw.FallbackWeek = dw
	return fvsw, fvsls, nil
}

func NewFeedVersionServiceWindowFromReader(reader tl.Reader) (FeedVersionServiceWindow, error) {
	fvsw := FeedVersionServiceWindow{}
	// Get Feed Info dates
	if start, end, err := feedDates(reader); err == nil {
		fvsw.FeedStartDate = start
		fvsw.FeedEndDate = end
	} else {
		return fvsw, err
	}
	// Recalculate service bounds
	if start, end, err := tl.FeedVersionServiceBounds(reader); err == nil {
		if !start.IsZero() && !end.IsZero() {
			fvsw.EarliestCalendarDate = tt.NewDate(start)
			fvsw.LatestCalendarDate = tt.NewDate(end)
		}
	} else {
		return fvsw, err
	}
	// Get the default timezone.
	// Spec requires this field and all values to be identical.
	for ent := range reader.Agencies() {
		if tz, ok := tt.IsValidTimezone(ent.AgencyTimezone); ok {
			fvsw.DefaultTimezone = tt.NewString(tz)
		}
		break
	}
	return fvsw, nil
}

type FeedVersionServiceWindow struct {
	FeedVersionID        int
	FeedStartDate        tt.Date
	FeedEndDate          tt.Date
	EarliestCalendarDate tt.Date
	LatestCalendarDate   tt.Date
	FallbackWeek         tt.Date
	DefaultTimezone      tt.String
	tl.DatabaseEntity
	tl.Timestamps
}

func (fvi *FeedVersionServiceWindow) EntityID() string {
	return strconv.Itoa(fvi.ID)
}

func (FeedVersionServiceWindow) TableName() string {
	return "feed_version_service_windows"
}

func feedDates(reader tl.Reader) (tt.Date, tt.Date, error) {
	start := tt.Date{}
	end := tt.Date{}
	for fi := range reader.FeedInfos() {
		start = fi.FeedStartDate
		end = fi.FeedEndDate
		break
	}
	return start, end, nil
}
