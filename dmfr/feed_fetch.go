package dmfr

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
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
	FeedVersionID tt.Int // optional field, don't use tl.FeedVersionEntity
	tl.Timestamps
	tl.DatabaseEntity
}

func (ent *FeedFetch) TableName() string {
	return "feed_fetches"
}
