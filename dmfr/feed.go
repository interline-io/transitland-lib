package dmfr

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/interline-io/transitland-lib/tt"
)

// Feed listed in a parsed DMFR file
type Feed struct {
	FeedID            string            `json:"id" db:"onestop_id"`
	SupersedesIDs     tt.Strings        `json:"supersedes_ids,omitempty" db:"-"`
	Name              tt.String         `json:"name,omitempty"`
	Description       tt.String         `json:"description,omitempty"`
	Spec              string            `json:"spec,omitempty"`
	URLs              FeedUrls          `json:"urls,omitempty" db:"urls"`
	Languages         FeedLanguages     `json:"languages,omitempty"`
	License           FeedLicense       `json:"license,omitempty"`
	Authorization     FeedAuthorization `json:"authorization,omitempty" db:"auth"`
	Tags              tt.Tags           `json:"tags,omitempty" db:"feed_tags" `
	File              string            `json:"-"` // internal
	DeletedAt         tt.Time           `json:"-"` // internal
	tt.DatabaseEntity `json:"-"`        // internal
	tt.Timestamps     `json:"-"`        // internal
}

func (ent *Feed) MatchSecrets(secrets []Secret, urltype string) (Secret, error) {
	found := Secret{}
	count := 0
	for _, secret := range secrets {
		// Filter on URL type if provided
		if urltype != "" && secret.URLType != "" && secret.URLType != urltype {
			continue
		}
		if secret.MatchFeed(ent.FeedID) {
			count += 1
			found = secret
		}
	}
	// Try matching on filename if no exact feed id matches were found
	if count == 0 {
		for _, secret := range secrets {
			if urltype != "" && secret.URLType != "" && secret.URLType != urltype {
				continue
			}
			if secret.MatchFilename(ent.File) {
				count += 1
				found = secret
			}
		}
	}
	if count == 0 {
		return Secret{}, errors.New("no matching secret found")
	} else if count > 1 {
		return Secret{}, fmt.Errorf("ambiguous secrets; %d matches", count)
	}
	return found, nil
}

// Equal compares the JSON representation of two feeds
func (ent *Feed) Equal(other *Feed) bool {
	if other == nil {
		return false
	}
	a1 := *ent
	a2 := *other
	a1j, _ := json.Marshal(&a1)
	a2j, _ := json.Marshal(&a2)
	if ent.File != other.File {
		return false
	}
	return string(a1j) == string(a2j)
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
	StaticPlanned            []string `json:"static_planned,omitempty"`
	StaticHistoric           []string `json:"static_historic,omitempty"`
	StaticHypothetical       []string `json:"static_hypothetical,omitempty"`
	RealtimeVehiclePositions string   `json:"realtime_vehicle_positions,omitempty"`
	RealtimeTripUpdates      string   `json:"realtime_trip_updates,omitempty"`
	RealtimeAlerts           string   `json:"realtime_alerts,omitempty"`
	GbfsAutoDiscovery        string   `json:"gbfs_auto_discovery,omitempty"`
	GbfsSystemAlerts         string   `json:"gbfs_system_alerts,omitempty"`
	MdsProvider              string   `json:"mds_provider,omitempty"`
	// GbfsSystemInformation string
	// GbfsStationInformation string
	// GbfsStationStatus      string
	// GbfsFreeBikeStatus     string
	// GbfsSystemHours        string
	// GbfsSystemCalendar     string
	// GbfsSystemRegions      string
	// GbfsSystemPricingPlans string
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
	Type      string `json:"type,omitempty"` // ["header", "basic_auth", "query_param", "path_segment", "replace_url"]
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
type FeedAssociatedFeeds []string

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
