package dmfr

import (
	"database/sql"
	"strconv"

	tl "github.com/interline-io/transitland-lib"
)

// FeedState .
type FeedState struct {
	ID                    int
	FeedID                int
	FeedVersionID         tl.OptionalKey
	LastFetchError        string
	LastFetchedAt         tl.OptionalTime
	LastSuccessfulFetchAt tl.OptionalTime
	FeedPriority          sql.NullInt64
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

// TableName .
func (ent *FeedState) TableName() string {
	return "feed_states"
}
