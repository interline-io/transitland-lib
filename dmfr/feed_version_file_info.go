package dmfr

import (
	"strconv"

	"github.com/interline-io/transitland-lib/tt"
)

// FeedVersionFileInfo .
type FeedVersionFileInfo struct {
	Name         string
	Size         int64
	Rows         int64
	Columns      int
	Header       string
	CSVLike      bool
	SHA1         string
	ValuesUnique tt.Counts
	ValuesCount  tt.Counts
	tt.FeedVersionEntity
	tt.DatabaseEntity
	tt.Timestamps
}

// EntityID .
func (fvi *FeedVersionFileInfo) EntityID() string {
	return strconv.Itoa(fvi.ID)
}

// TableName .
func (FeedVersionFileInfo) TableName() string {
	return "feed_version_file_infos"
}
