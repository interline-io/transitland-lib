package dmfr

import (
	"strconv"

	"github.com/interline-io/transitland-lib/tl"
)

// FeedState .
type FeedState struct {
	ID                    int
	FeedID                int
	FeedVersionID         tl.Int
	LastFetchError        string
	LastFetchedAt         tl.Time
	LastSuccessfulFetchAt tl.Time
	FeedPriority          tl.Int
	FeedRealtimeEnabled   bool
	tl.Timestamps
}

// EntityID .
func (ent *FeedState) EntityID() string {
	return strconv.Itoa(ent.ID)
}

// SetID .
func (ent *FeedState) SetID(id int) {
	ent.ID = id
}

// GetID .
func (ent *FeedState) GetID() int {
	return ent.ID
}

// TableName .
func (ent *FeedState) TableName() string {
	return "feed_states"
}
