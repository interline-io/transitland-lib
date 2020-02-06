package gotransit

import (
	"errors"
	"time"
)

type canSHA1 interface {
	SHA1() (string, error)
}

type canDirSHA1 interface {
	DirSHA1() (string, error)
}

type canPath interface {
	Path() string
}

// FeedVersion represents a single GTFS data source.
type FeedVersion struct {
	ID                   int
	FeedID               int
	FeedType             string
	SHA1                 string
	SHA1Dir              string
	File                 string
	URL                  string
	EarliestCalendarDate time.Time
	LatestCalendarDate   time.Time
	FetchedAt            time.Time
	Timestamps
}

// EntityID .
func (ent *FeedVersion) EntityID() string {
	return entID(ent.ID, "0")
}

// TableName sets the table name prefix.
func (ent *FeedVersion) TableName() string {
	return "feed_versions"
}

// NewFeedVersionFromReader returns a FeedVersion from a Reader.
func NewFeedVersionFromReader(reader Reader) (FeedVersion, error) {
	fv := FeedVersion{}
	fv.FeedType = "Feed" // tlv1 compat
	// Perform basic GTFS validity checks
	if errs := reader.ValidateStructure(); len(errs) > 0 {
		return fv, errs[0]
	}
	// Get service dates
	start, end, err := servicePeriod(reader)
	if err != nil {
		return fv, err
	}
	fv.EarliestCalendarDate = start
	fv.LatestCalendarDate = end
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

func servicePeriod(reader Reader) (time.Time, time.Time, error) {
	var start time.Time
	var end time.Time
	for c := range reader.Calendars() {
		if start.IsZero() || c.StartDate.Before(start) {
			start = c.StartDate
		}
		if end.IsZero() || c.EndDate.After(end) {
			end = c.EndDate
		}
	}
	for cd := range reader.CalendarDates() {
		if cd.ExceptionType != 1 {
			continue
		}
		if start.IsZero() || cd.Date.Before(start) {
			start = cd.Date
		}
		if end.IsZero() || cd.Date.After(end) {
			end = cd.Date
		}
	}
	if start.IsZero() || end.IsZero() {
		return start, end, errors.New("start or end dates were empty")
	}
	if end.Before(start) {
		return start, end, errors.New("end before start")
	}
	return start, end, nil
}
