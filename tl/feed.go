package tl

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strconv"
)

// Feed listed in a parsed DMFR file
type Feed struct {
	ID              int                 `json:"-"`
	FeedID          string              `json:"id" db:"onestop_id"`
	FeedNamespaceID string              `json:"feed_namespace_id"`
	Spec            string              `json:"spec"`
	URLs            FeedUrls            `json:"urls" db:"urls"`
	AssociatedFeeds FeedAssociatedFeeds `json:"-"` // `json:"associated_feeds"`
	Languages       FeedLanguages       `json:"languages,omitempty"`
	License         FeedLicense         `json:"license"`
	Authorization   FeedAuthorization   `json:"authorization" db:"auth"`
	OtherIDs        map[string]string   `json:"other_ids" db:"-"`
	IDCrosswalk     map[string]string   `json:"id_crosswalk" db:"-"`
	File            string              `json:"-"` // internal
	DeletedAt       OptionalTime        `json:"-"` // internal
	Timestamps      `json:"-"`          // internal
}

// SetID .
func (ent *Feed) SetID(id int) {
	ent.ID = id
}

// GetID .
func (ent *Feed) GetID() int {
	return ent.ID
}

// EntityID .
func (ent *Feed) EntityID() string {
	return strconv.Itoa(ent.ID)
}

// TableName .
func (Feed) TableName() string {
	return "current_feeds"
}

// FeedUrls contains URL values for a Feed.
type FeedUrls struct {
	StaticCurrent            string   `json:"static_current,omitempty"`
	StaticHistoric           []string `json:"static_historic,omitempty"`
	StaticPlanned            string   `json:"static_planner,omitempty"`
	RealtimeVehiclePositions string   `json:"realtime_vehicle_positions,omitempty"`
	RealtimeTripUpdates      string   `json:"realtime_trip_updates,omitempty"`
	RealtimeAlerts           string   `json:"realtime_alerts,omitempty"`
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
	SpdxIdentifier          string `json:"spdx_identifier,omitempty"`
	URL                     string `json:"url,omitempty"`
	UseWithoutAttribution   string `json:"use_without_attribution,omitempty"`
	CreateDerivedProduct    string `json:"create_derived_product,omitempty"`
	RedistributionAllowed   string `json:"redistribution_allowed,omitempty"`
	CommercialUseAllowed    string `json:"commercial_use_allowed,omitempty"`
	ShareAlikeOptional      string `json:"share_alike_optional,omitempty"`
	AttributionText         string `json:"attribution_text,omitempty"`
	AttributionInstructions string `json:"attribution_instructions,omitempty"`
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
	Type      string `json:"type,omitempty"` // ["header", "basic_auth", "query_param", "path_segment"]
	ParamName string `json:"param_name,omitempty"`
	InfoURL   string `json:"info_url,omitempty"`
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
type FeedLanguages []string

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
