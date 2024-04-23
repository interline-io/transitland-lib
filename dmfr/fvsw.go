package dmfr

import (
	"strconv"

	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
)

func NewFeedVersionServiceWindowFromReader(reader tl.Reader) (FeedVersionServiceWindow, error) {
	ret := FeedVersionServiceWindow{}
	fvswBuilder := NewFeedVersionServiceWindowBuilder()
	if err := copier.QuietCopy(reader, &empty.Writer{}, func(o *copier.Options) { o.AddExtension(fvswBuilder) }); err != nil {
		return ret, err
	}
	ret, err := fvswBuilder.ServiceWindow()
	if err != nil {
		return ret, err
	}
	return ret, nil
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

type FeedVersionServiceWindowBuilder struct {
	fvsw FeedVersionServiceWindow
}

func NewFeedVersionServiceWindowBuilder() *FeedVersionServiceWindowBuilder {
	return &FeedVersionServiceWindowBuilder{}
}

func (pp *FeedVersionServiceWindowBuilder) AfterWrite(eid string, ent tl.Entity, emap *tl.EntityMap) error {
	switch v := ent.(type) {
	case *tl.Agency:
		if tz, ok := tt.IsValidTimezone(v.AgencyTimezone); ok {
			pp.fvsw.DefaultTimezone = tt.NewString(tz)
		}
	case *tl.FeedInfo:
		pp.fvsw.FeedStartDate = v.FeedStartDate
		pp.fvsw.FeedEndDate = v.FeedEndDate
	case *tl.Service:
		cStart, cEnd := v.ServicePeriod()
		retStart, retEnd := pp.fvsw.EarliestCalendarDate.Val, pp.fvsw.LatestCalendarDate.Val
		if retStart.IsZero() || cStart.Before(retStart) {
			pp.fvsw.EarliestCalendarDate = tt.NewDate(cStart)
		}
		if retEnd.IsZero() || cEnd.After(retEnd) {
			pp.fvsw.LatestCalendarDate = tt.NewDate(cEnd)
		}
	}
	return nil
}

func (pp *FeedVersionServiceWindowBuilder) Copy(*copier.Copier) error {
	return nil
}

func (pp *FeedVersionServiceWindowBuilder) ServiceWindow() (FeedVersionServiceWindow, error) {
	return pp.fvsw, nil
}
