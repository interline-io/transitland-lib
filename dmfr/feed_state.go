package dmfr

import (
	"strconv"

	"github.com/interline-io/transitland-lib/tl"
)

// FeedState .
type FeedState struct {
	ID                    int
	FeedID                int
	FeedVersionID         tl.OInt
	LastFetchError        string
	LastFetchedAt         tl.OptionalTime
	LastSuccessfulFetchAt tl.OptionalTime
	FeedPriority          tl.OInt
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
