package stats

import (
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tt"
)

type FeedVersionServiceWindowBuilder struct {
	fvsw dmfr.FeedVersionServiceWindow
}

func NewFeedVersionServiceWindowFromReader(reader adapters.Reader) (dmfr.FeedVersionServiceWindow, error) {
	ret := dmfr.FeedVersionServiceWindow{}
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

func NewFeedVersionServiceWindowBuilder() *FeedVersionServiceWindowBuilder {
	return &FeedVersionServiceWindowBuilder{}
}

func (pp *FeedVersionServiceWindowBuilder) AfterWrite(eid string, ent tt.Entity, emap *tt.EntityMap) error {
	switch v := ent.(type) {
	case *gtfs.Agency:
		if tz, ok := tt.IsValidTimezone(v.AgencyTimezone.Val); ok {
			pp.fvsw.DefaultTimezone.Set(tz)
		}
	case *gtfs.FeedInfo:
		pp.fvsw.FeedStartDate = v.FeedStartDate
		pp.fvsw.FeedEndDate = v.FeedEndDate
	case *gtfs.Calendar:
		svc := service.NewService(*v, v.CalendarDates...)
		cStart, cEnd := svc.ServicePeriod()
		retStart, retEnd := pp.fvsw.EarliestCalendarDate.Val, pp.fvsw.LatestCalendarDate.Val
		if retStart.IsZero() || cStart.Before(retStart) {
			pp.fvsw.EarliestCalendarDate.Set(cStart)
		}
		if retEnd.IsZero() || cEnd.After(retEnd) {
			pp.fvsw.LatestCalendarDate.Set(cEnd)
		}
	}
	return nil
}

func (pp *FeedVersionServiceWindowBuilder) Copy(adapters.EntityCopier) error {
	return nil
}

func (pp *FeedVersionServiceWindowBuilder) ServiceWindow() (dmfr.FeedVersionServiceWindow, error) {
	return pp.fvsw, nil
}
