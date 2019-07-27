package gotransit

import (
	"sort"
	"time"
)

// FeedVersion represents a single GTFS data source.
type FeedVersion struct {
	ID                   int
	FeedID               int
	FeedType             string
	SHA1                 string
	File                 string
	URL                  string
	EarliestCalendarDate time.Time
	LatestCalendarDate   time.Time
	FetchedAt            time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// TableName sets the table name prefix.
func (ent *FeedVersion) TableName() string {
	return "feed_versions"
}

// NewFeedVersionFromReader returns a FeedVersion from a Reader.
func NewFeedVersionFromReader(reader Reader) *FeedVersion {
	fv := FeedVersion{}
	// Get Calendar Dates
	times := []time.Time{}
	for c := range reader.Calendars() {
		times = append(times, c.StartDate)
		times = append(times, c.EndDate)
	}
	for c := range reader.CalendarDates() {
		if c.ExceptionType == 1 {
			times = append(times, c.Date)
		}
	}
	sort.Slice(times, func(i, j int) bool {
		return times[i].Before(times[j])
	})
	if len(times) > 0 {
		if times[0].Before(times[len(times)-1]) {
			fv.EarliestCalendarDate = times[0]
			fv.LatestCalendarDate = times[len(times)-1]
		}
	}
	type canSHA1 interface {
		SHA1() (string, error)
	}
	if s, ok := reader.(canSHA1); ok {
		if h, err := s.SHA1(); err == nil {
			fv.SHA1 = h
		}
	}
	return &fv
}
