package gotransit

import (
	"fmt"
	"time"

	"github.com/interline-io/gotransit/causes"
)

// FeedInfo feed_info.txt
type FeedInfo struct {
	FeedPublisherName string    `csv:"feed_publisher_name" required:"true" gorm:"not null"`
	FeedPublisherURL  string    `csv:"feed_publisher_url" required:"true" validator:"url" gorm:"not null"`
	FeedLang          string    `csv:"feed_lang" validator:"lang" required:"true" gorm:"not null"`
	FeedStartDate     time.Time `csv:"feed_start_date"`
	FeedEndDate       time.Time `csv:"feed_end_date"`
	FeedVersion       string    `csv:"feed_version"`
	BaseEntity
}

// EntityID returns nothing.
func (ent *FeedInfo) EntityID() string {
	return ""
}

// Warnings for this Entity.
func (ent *FeedInfo) Warnings() (errs []error) {
	return errs
}

// Errors for this Entity.
func (ent *FeedInfo) Errors() (errs []error) {
	errs = ValidateTags(ent)
	errs = append(errs, ent.BaseEntity.loadErrors...)
	if ent.FeedEndDate.Before(ent.FeedStartDate) {
		errs = append(errs, causes.NewInvalidFieldError("feed_end_date", "", fmt.Errorf("feed_end_date '%s' must come after feed_start_date '%s'", ent.FeedEndDate, ent.FeedStartDate)))

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
