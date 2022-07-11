package tl

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/enum"
)

// FeedInfo feed_info.txt
type FeedInfo struct {
	FeedPublisherName String
	FeedPublisherURL  Url
	FeedVersion       String `db:"feed_version_name"`
	FeedLang          Language
	FeedStartDate     Date
	FeedEndDate       Date
	DefaultLang       Language
	FeedContactEmail  Email
	FeedContactURL    Url
	BaseEntity
}

// Errors for this Entity.
func (ent *FeedInfo) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	if ent.FeedStartDate.Val.IsZero() || ent.FeedEndDate.Val.IsZero() {
		// skip
	} else {
		if ent.FeedEndDate.Val.Before(ent.FeedStartDate.Val) {
			errs = append(errs,
				causes.NewInvalidFieldError(
					"feed_end_date",
					"",
					fmt.Errorf("feed_end_date '%s' must come after feed_start_date '%s'", ent.FeedEndDate.Val, ent.FeedStartDate.Val)))
		}
	}
	errs = enum.CheckErrors(
		errs,
		enum.CheckFieldPresentError("feed_lang", &ent.FeedLang),
		enum.CheckFieldPresentError("feed_publisher_url", &ent.FeedPublisherURL),
		enum.CheckFieldError("default_lang", &ent.DefaultLang),
		enum.CheckFieldError("feed_contact_email", &ent.FeedContactEmail),
		enum.CheckFieldError("feed_contact_url", &ent.FeedContactURL),
		enum.CheckFieldPresent("feed_publisher_name", &ent.FeedPublisherName),
	)
	return errs
}

// Filename feed_info.txt
func (ent *FeedInfo) Filename() string {
	return "feed_info.txt"
}

// TableName gtfs_feed_infos
func (ent *FeedInfo) TableName() string {
	return "gtfs_feed_infos"
}
