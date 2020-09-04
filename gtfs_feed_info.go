package tl

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/enums"
)

// FeedInfo feed_info.txt
type FeedInfo struct {
	FeedPublisherName string       `csv:"feed_publisher_name" required:"true"`
	FeedPublisherURL  string       `csv:"feed_publisher_url" required:"true"`
	FeedLang          string       `csv:"feed_lang" required:"true"`
	FeedStartDate     OptionalTime `csv:"feed_start_date"`
	FeedEndDate       OptionalTime `csv:"feed_end_date"`
	FeedVersion       string       `csv:"feed_version" db:"feed_version_name"`
	BaseEntity
}

// EntityID returns nothing.
func (ent *FeedInfo) EntityID() string {
	return ""
}

// Errors for this Entity.
func (ent *FeedInfo) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, enums.CheckPresent("feed_publisher_name", ent.FeedPublisherName)...)
	errs = append(errs, enums.CheckPresent("feed_publisher_url", ent.FeedPublisherURL)...)
	errs = append(errs, enums.CheckPresent("feed_lang", ent.FeedLang)...)
	errs = append(errs, enums.CheckURL("feed_publisher_url", ent.FeedPublisherURL)...)
	errs = append(errs, enums.CheckLanguage("feed_lang", ent.FeedLang)...)
	if ent.FeedStartDate.IsZero() && ent.FeedEndDate.IsZero() {
		// skip
	} else {
		if ent.FeedEndDate.Time.Before(ent.FeedStartDate.Time) {
			errs = append(errs, causes.NewInvalidFieldError("feed_end_date", "", fmt.Errorf("feed_end_date '%s' must come after feed_start_date '%s'", ent.FeedEndDate.Time, ent.FeedStartDate.Time)))
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
