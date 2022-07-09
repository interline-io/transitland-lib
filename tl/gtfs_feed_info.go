package tl

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/enum"
)

// FeedInfo feed_info.txt
type FeedInfo struct {
	FeedPublisherName string   `csv:",required"`
	FeedPublisherURL  string   `csv:",required"`
	FeedLang          Language `csv:",required"`
	FeedVersion       string   `db:"feed_version_name"`
	FeedStartDate     Date
	FeedEndDate       Date
	DefaultLang       Language
	FeedContactEmail  String
	FeedContactURL    String
	BaseEntity
}

// Errors for this Entity.
func (ent *FeedInfo) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, enum.CheckPresent("feed_publisher_name", ent.FeedPublisherName)...)
	errs = append(errs, enum.CheckPresent("feed_publisher_url", ent.FeedPublisherURL)...)
	errs = enum.CheckError(errs, enum.CheckFieldPresentError("feed_lang", &ent.FeedLang))
	errs = enum.CheckError(errs, enum.CheckFieldError("default_lang", &ent.DefaultLang))

	errs = append(errs, enum.CheckURL("feed_publisher_url", ent.FeedPublisherURL)...)
	errs = append(errs, enum.CheckEmail("feed_contact_email", ent.FeedContactEmail.String)...)
	errs = append(errs, enum.CheckURL("feed_contact_url", ent.FeedContactURL.String)...)
	if ent.FeedStartDate.Val.IsZero() || ent.FeedEndDate.Val.IsZero() {
		// skip
	} else {
		if ent.FeedEndDate.Val.Before(ent.FeedStartDate.Val) {
			errs = append(errs, causes.NewInvalidFieldError("feed_end_date", "", fmt.Errorf("feed_end_date '%s' must come after feed_start_date '%s'", ent.FeedEndDate.Val, ent.FeedStartDate.Val)))
		}
	}
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
