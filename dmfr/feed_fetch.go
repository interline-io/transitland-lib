package dmfr

import (
	"github.com/interline-io/transitland-lib/tt"
)

type FeedFetch struct {
	FeedID        int
	URLType       string
	URL           string
	Success       bool
	FetchedAt     tt.Time
	FetchError    tt.String
	ResponseSize  tt.Int
	ResponseCode  tt.Int
	ResponseSHA1  tt.String
	FeedVersionID tt.Int // optional field, don't use tt.FeedVersionEntity
	tt.Timestamps
	tt.DatabaseEntity
}

func (ent *FeedFetch) TableName() string {
	return "feed_fetches"
}
