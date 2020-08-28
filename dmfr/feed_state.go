package dmfr

import (
	"database/sql"
	"strconv"

	"github.com/interline-io/gotransit"
)

// FeedState .
type FeedState struct {
	ID                    int
	FeedID                int
	FeedVersionID         gotransit.OptionalKey
	LastFetchError        string
	LastFetchedAt         gotransit.OptionalTime
	LastSuccessfulFetchAt gotransit.OptionalTime
	FeedPriority          sql.NullInt64
	FeedRealtimeEnabled   bool
	gotransit.Timestamps
}

// EntityID .
func (ent *FeedState) EntityID() string {
	return strconv.Itoa(ent.ID)
}

// TableName .
func (ent *FeedState) TableName() string {
	return "feed_states"
}
