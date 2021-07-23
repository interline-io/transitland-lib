package tl

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

type Operator struct {
	ID              int                     `json:"-"`
	OnestopID       OString                 `json:"onestop_id"`
	Tags            Tags                    `json:"tags" db:"operator_tags"`
	Name            OString                 `json:"name"`
	ShortName       OString                 `json:"short_name"`
	Website         OString                 `json:"website"`
	AssociatedFeeds OperatorAssociatedFeeds `json:"associated_feeds"`
	DeletedAt       OTime                   `json:"-"`
	Timestamps      `json:"-"`
}

// Equal compares the JSON representation of two operators.
func (ent *Operator) Equal(other *Operator) bool {
	if other == nil {
		return false
	}
	a1 := *ent
	a2 := *other
	a1j, _ := json.Marshal(&a1)
	a2j, _ := json.Marshal(&a2)
	return string(a1j) == string(a2j)
}

// TableName .
func (Operator) TableName() string {
	return "current_operators"
}

// SetID .
func (ent *Operator) SetID(id int) {
	ent.ID = id
}

// GetID .
func (ent *Operator) GetID() int {
	return ent.ID
}

////////////

type OperatorAssociatedFeed struct {
	FeedOnestopID OString `json:"feed_onestop_id,omitempty"`
	GtfsAgencyID  OString `json:"gtfs_agency_id,omitempty" db:"gtfs_agency_id"`
	AgencyID      OInt    `json:"-"` // internal
	ID            int     `json:"-"` // internal
	FeedID        int     `json:"-"` // internal
}

// OperatorAssociatedFeeds is necessary to scan correctly from database
type OperatorAssociatedFeeds []OperatorAssociatedFeed

// Value .
func (a OperatorAssociatedFeeds) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan .
func (a *OperatorAssociatedFeeds) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	err := json.Unmarshal(b, &a)
	if err != nil {
		return err
	}
	return nil
}
