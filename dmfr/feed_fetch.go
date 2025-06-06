package dmfr

import (
	"github.com/interline-io/transitland-lib/tt"
)

// FeedFetch is a record of when feed data was fetched via a URL
type FeedFetch struct {
	FeedID               int
	URLType              string
	URL                  string
	Success              bool
	FetchedAt            tt.Time
	FetchError           tt.String
	ResponseSize         tt.Int
	ResponseCode         tt.Int
	ResponseTtfbMs       tt.Int
	ResponseTimeMs       tt.Int
	ResponseSHA1         tt.String
	FeedVersionID        tt.Int // optional field, don't use FeedVersionEntity
	ValidationDurationMs tt.Int
	UploadDurationMs     tt.Int
	tt.Timestamps
	tt.DatabaseEntity
}

func (ent *FeedFetch) TableName() string {
	return "feed_fetches"
}
