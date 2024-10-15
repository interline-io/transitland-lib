package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// FeedInfo feed_info.txt
type FeedInfo struct {
	FeedPublisherName tt.String   `csv:",required"`
	FeedPublisherURL  tt.Url      `csv:",required"`
	FeedLang          tt.Language `csv:",required"`
	FeedVersion       tt.String   `db:"feed_version_name"`
	FeedStartDate     tt.Date
	FeedEndDate       tt.Date
	DefaultLang       tt.Language
	FeedContactEmail  tt.Email
	FeedContactURL    tt.Url
	tt.BaseEntity
}

// Filename feed_info.txt
func (ent *FeedInfo) Filename() string {
	return "feed_info.txt"
}

// TableName gtfs_feed_infos
func (ent *FeedInfo) TableName() string {
	return "gtfs_feed_infos"
}

// Errors for this Entity.
func (ent *FeedInfo) ConditionalErrors() (errs []error) {
	if ent.FeedStartDate.IsZero() || ent.FeedEndDate.IsZero() {
		// skip
	} else if ent.FeedEndDate.Val.Before(ent.FeedStartDate.Val) {
		errs = append(errs, causes.NewInvalidFieldError("feed_end_date", ent.FeedStartDate.Val.String(), fmt.Errorf("feed_end_date '%s' must come after feed_start_date '%s'", ent.FeedEndDate.Val, ent.FeedStartDate.Val)))
	}
	return errs
}
