package tl

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

// FeedInfo feed_info.txt
type FeedInfo struct {
	FeedPublisherName string `csv:",required"`
	FeedPublisherURL  string `csv:",required"`
	FeedLang          string `csv:",required"`
	FeedVersion       string `db:"feed_version_name"`
	FeedStartDate     tt.Date
	FeedEndDate       tt.Date
	DefaultLang       tt.String
	FeedContactEmail  tt.String
	FeedContactURL    tt.String
	BaseEntity
}

// Errors for this Entity.
func (ent *FeedInfo) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, tt.CheckPresent("feed_publisher_name", ent.FeedPublisherName)...)
	errs = append(errs, tt.CheckPresent("feed_publisher_url", ent.FeedPublisherURL)...)
	errs = append(errs, tt.CheckPresent("feed_lang", ent.FeedLang)...)
	errs = append(errs, tt.CheckURL("feed_publisher_url", ent.FeedPublisherURL)...)
	errs = append(errs, tt.CheckLanguage("feed_lang", ent.FeedLang)...)
	errs = append(errs, tt.CheckLanguage("default_lang", ent.DefaultLang.Val)...)
	errs = append(errs, tt.CheckEmail("feed_contact_email", ent.FeedContactEmail.Val)...)
	errs = append(errs, tt.CheckURL("feed_contact_url", ent.FeedContactURL.Val)...)
	if ent.FeedStartDate.IsZero() || ent.FeedEndDate.IsZero() {
		// skip
	} else {
		if ent.FeedEndDate.Val.Before(ent.FeedStartDate.Val) {
			errs = append(errs, causes.NewInvalidFieldError("feed_end_date", ent.FeedStartDate.Val.String(), fmt.Errorf("feed_end_date '%s' must come after feed_start_date '%s'", ent.FeedEndDate.Val, ent.FeedStartDate.Val)))
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
