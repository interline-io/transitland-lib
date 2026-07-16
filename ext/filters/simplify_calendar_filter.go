package filters

import (
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tt"
)

type SimplifyCalendarFilter struct {
}

func (e *SimplifyCalendarFilter) Filter(ent tt.Entity, _ *tt.EntityMap) error {
	v, ok := ent.(*gtfs.Calendar)
	if !ok {
		return nil
	}
	svc := service.NewService(*v, v.CalendarDates...)
	s, err := svc.Simplify()
	if err != nil {
		return err
	}
	v.CalendarDates = s.CalendarDates()
	v.StartDate.Set(s.StartDate.Val)
	v.EndDate.Set(s.EndDate.Val)
	v.Monday.Set(s.Monday.Val)
	v.Tuesday.Set(s.Tuesday.Val)
	v.Wednesday.Set(s.Wednesday.Val)
	v.Thursday.Set(s.Thursday.Val)
	v.Friday.Set(s.Friday.Val)
	v.Saturday.Set(s.Saturday.Val)
	v.Sunday.Set(s.Sunday.Val)
	return nil
}
