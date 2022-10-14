package dmfr

import (
	"strconv"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
)

func NewFeedVersionServiceWindowFromReader(reader tl.Reader) (FeedVersionServiceWindow, error) {
	fvsw := FeedVersionServiceWindow{}
	// Get Feed Info dates
	if start, end, err := feedDates(reader); err == nil {
		fvsw.FeedStartDate = start
		fvsw.FeedEndDate = end
	} else {
		return fvsw, err
	}
	return fvsw, nil
}

type FeedVersionServiceWindow struct {
	FeedVersionID int
	FeedStartDate tl.Date
	FeedEndDate   tl.Date
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
