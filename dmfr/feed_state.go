package dmfr

import (
	"strconv"

	"github.com/interline-io/transitland-lib/tl/tt"
)

// FeedState stores the pointer to the active FeedVersion.
type FeedState struct {
	FeedID              int
	FeedVersionID       tt.Int
	FeedPriority        tt.Int
	FetchWait           tt.Int
	FeedRealtimeEnabled bool
	Public              bool
	tt.DatabaseEntity
	tt.Timestamps
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
