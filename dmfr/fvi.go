package dmfr

import (
	"strconv"

	"github.com/interline-io/gotransit"
)

// FeedVersionImport .
type FeedVersionImport struct {
	ID            int
	FeedVersionID int
	ExceptionLog  string
	ImportLevel   int  // deprecated
	Success       bool // Finished, Success Yes/No
	InProgress    bool // In Progress
	gotransit.Timestamps
}

// EntityID .
func (fvi *FeedVersionImport) EntityID() string {
	return strconv.Itoa(fvi.ID)
}

// TableName .
func (FeedVersionImport) TableName() string {
	return "feed_version_gtfs_imports"
}
