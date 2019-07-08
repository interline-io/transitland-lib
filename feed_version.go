package gotransit

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"os"
	"sort"
	"time"
)

// FeedVersion represents a single GTFS data source.
type FeedVersion struct {
	FeedID               int    `gorm:"index"`
	FeedType             string `gorm:"index"`
	SHA1                 string `gorm:"index"`
	File                 string
	URL                  string
	EarliestCalendarDate OptionalTime `gorm:"index;not null"`
	LatestCalendarDate   OptionalTime `gorm:"index;not null"`
	FetchedAt            OptionalTime
	ID                   int
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// TableName sets the table name prefix.
func (ent *FeedVersion) TableName() string {
	return "feed_versions"
}

// NewFeedVersion returns a FeedVersion from a Reader.
func NewFeedVersion(reader Reader) (FeedVersion, error) {
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
			fv.EarliestCalendarDate.Time = times[0]
			fv.LatestCalendarDate.Time = times[len(times)-1]
		}
	}
	return fv, nil
}

// fileSHA1 returns the SHA1 hash of the zip file
func fileSHA1(path string) (string, error) {
	var sha1string string
	file, err := os.Open(path)
	if err != nil {
		return sha1string, err
	}
	defer file.Close()
	hash := sha1.New()
	if _, err := io.Copy(hash, file); err != nil {
		return sha1string, err
	}
	sha1bytes := hash.Sum(nil)[:20]
	sha1string = hex.EncodeToString(sha1bytes)
	return sha1string, nil
}
