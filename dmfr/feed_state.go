package dmfr

import (
	"strconv"

	"github.com/interline-io/transitland-lib/tt"
)

// FeedState holds a feed's active and materialized feed version pointers, its
// global-query visibility, and values that control feed fetch and permissions.
type FeedState struct {
	FeedID                    int
	FeedVersionID             tt.Int // transitional mirror of MaterializedFeedVersionID
	ActiveFeedVersionID       tt.Int // current version for retention/lifecycle; set even when excluded
	MaterializedFeedVersionID tt.Int // visible/materialized version; null when excluded from global
	ExcludeFromGlobal         bool   // keep this feed out of default global results and materialized tables
	FeedPriority              tt.Int
	FetchWait                 tt.Int
	FeedRealtimeEnabled       bool
	Public                    bool
	RTRetentionPeriod         int // days to retain archived RT messages; 0 disables
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
