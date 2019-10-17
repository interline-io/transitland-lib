package dmfr

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/interline-io/gotransit"
)

// Feed listed in a parsed DMFR file
type Feed struct {
	ID                    int    `json:"-"`
	FeedID                string `db:"onestop_id" json:"id"`
	FeedNamespaceID       string
	Spec                  string
	URL                   string
	URLs                  FeedUrls `db:"urls"`
	AssociatedFeeds       FeedAssociatedFeeds
	Languages             FeedLanguages
	License               FeedLicense
	Authorization         FeedAuthorization `db:"-"`
	OtherIDs              map[string]string `db:"-" json:"other_ids"`
	IDCrosswalk           map[string]string `db:"-" json:"id_crosswalk"`
	LastFetchedAt         time.Time
	LastSuccessfulFetchAt time.Time
	LastFetchError        string
	gotransit.Timestamps
}

// TableName .
func (Feed) TableName() string {
	return "current_feeds"
}

// FeedUrls contains URL values for a Feed.
type FeedUrls struct {
	StaticCurrent            string
	StaticHistoric           string
	StaticPlanned            string
	RealtimeVehiclePositions string
	RealtimeTripUpdates      string
	RealtimeAlerts           string
	// StaticHypothetical    string
	// GbfsSystemInformation string
	// GbfsStationInformation string
	// GbfsStationStatus      string
	// GbfsFreeBikeStatus     string
	// GbfsSystemHours        string
	// GbfsSystemCalendar     string
	// GbfsSystemRegions      string
	// GbfsSystemPricingPlans string
	// GbfsSystemAlerts       string
	// MdsProvider            string
}

// Value .
func (a FeedUrls) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan .
func (a *FeedUrls) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

// FeedLicense describes the license and usage information for a Feed.
type FeedLicense struct {
	SpdxIdentifier          string
	URL                     string
	UseWithoutAttribution   string
	CreateDerivedProduct    string
	RedistributionAllowed   string
	CommercialUseAllowed    string
	ShareAlikeOptional      string
	AttributionText         string
	AttributionInstructions string
}

// Value .
func (a FeedLicense) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan .
func (a *FeedLicense) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

// FeedAuthorization contains details about how to access a Feed.
type FeedAuthorization struct {
	Type      string // ["header", "basic_auth", "query_param", "path_segment"]
	ParamName string
	InfoURL   string
}

// Value .
func (a FeedAuthorization) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan .
func (a *FeedAuthorization) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

// FeedAssociatedFeeds .
type FeedAssociatedFeeds map[string]string

// Value .
func (a FeedAssociatedFeeds) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan .
func (a *FeedAssociatedFeeds) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

// FeedLanguages .
type FeedLanguages map[string]string

// Value .
func (a FeedLanguages) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan .
func (a *FeedLanguages) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}
