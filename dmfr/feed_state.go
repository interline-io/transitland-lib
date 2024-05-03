package dmfr

import (
	"database/sql"
	"strconv"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
	"github.com/interline-io/transitland-lib/tldb"
)

// FeedState stores the pointer to the active FeedVersion.
type FeedState struct {
	FeedID        int
	FeedVersionID tt.Int
	FeedPriority  tt.Int
	FetchWait     tt.Int
	Public        bool
	tl.DatabaseEntity
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

func GetFeedState(atx tldb.Adapter, feedId int) (FeedState, error) {
	// Get state, create if necessary
	fs := FeedState{FeedID: feedId}
	if err := atx.Get(&fs, `SELECT * FROM feed_states WHERE feed_id = ?`, feedId); err == sql.ErrNoRows {
		fs.ID, err = atx.Insert(&fs)
		if err != nil {
			return fs, err
		}
	} else if err != nil {
		return fs, err
	}
	return fs, nil
}
