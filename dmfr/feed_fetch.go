package dmfr

import "github.com/interline-io/transitland-lib/tl"

type FeedFetch struct {
	ID            int
	FeedID        int
	URLType       string
	URL           string
	Success       bool
	FetchedAt     tl.Time
	FetchError    tl.String
	ResponseSize  tl.Int
	ResponseCode  tl.Int
	ResponseSHA1  tl.String
	FeedVersionID tl.Int
	tl.Timestamps
	tl.DatabaseEntity
}

func (ent *FeedFetch) TableName() string {
	return "feed_fetches"
}
