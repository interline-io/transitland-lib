package dmfr

import (
	"strconv"

	"github.com/interline-io/transitland-lib/tt"
)

// FeedState stores a pointer to the active FeedVersion and values that control feed fetch and permissions.
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

// TableName .
func (ent *FeedState) TableName() string {
	return "feed_states"
}
