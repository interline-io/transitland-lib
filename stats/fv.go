package stats

import (
	"errors"
	"time"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tt"
)

// NewFeedVersionFromReader returns a FeedVersion from a Reader.
func NewFeedVersionFromReader(reader adapters.Reader) (dmfr.FeedVersion, error) {
	fv := dmfr.FeedVersion{}
	// Perform basic GTFS validity checks
	if errs := reader.ValidateStructure(); len(errs) > 0 {
		return fv, errs[0]
	}
	// Get service dates
	if start, end, err := FeedVersionServiceBounds(reader); err == nil {
		fv.EarliestCalendarDate.Set(start)
		fv.LatestCalendarDate.Set(end)
	} else {
		return fv, err
	}
	// Get path and sha1
	if s, ok := reader.(canSHA1); ok {
		if h, err := s.SHA1(); err == nil {
			fv.SHA1 = h
		}
	}
	if s, ok := reader.(canDirSHA1); ok {
		if h, err := s.DirSHA1(); err == nil {
			fv.SHA1Dir = h
		}
	}
	if s, ok := reader.(canPath); ok {
		fv.File = s.Path()
	}
	return fv, nil
}

type canSHA1 interface {
	SHA1() (string, error)
}

type canDirSHA1 interface {
	DirSHA1() (string, error)
}

type canPath interface {
	Path() string
}

func FeedVersionServiceBounds(reader adapters.Reader) (time.Time, time.Time, error) {
	var start tt.Date
	var end tt.Date
	for c := range reader.Calendars() {
		if start.IsZero() || c.StartDate.Before(start) {
			start.Set(c.StartDate.Val)
		}
		if end.IsZero() || c.EndDate.After(end) {
			end.Set(c.EndDate.Val)
		}
	}
	for cd := range reader.CalendarDates() {
		if cd.ExceptionType.Val != 1 {
			continue
		}
		if start.IsZero() || cd.Date.Before(start) {
			start.Set(cd.Date.Val)
		}
		if end.IsZero() || cd.Date.After(end) {
			end.Set(cd.Date.Val)
		}
	}
	if start.IsZero() || end.IsZero() {
		return start.Val, end.Val, errors.New("start or end dates were empty")
	}
	if end.Before(start) {
		return start.Val, end.Val, errors.New("end before start")
	}
	return start.Val, end.Val, nil
}
