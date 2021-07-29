package tl

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"
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
	GtfsAgencyID         OString `json:"gtfs_agency_id,omitempty" db:"gtfs_agency_id"`
	FeedOnestopID        OString `json:"feed_onestop_id,omitempty" db:"-"`
	ResolvedOnestopID    OString `json:"-"` // internal
	ResolvedGtfsAgencyID OString `json:"-"` // internal
	ResolvedName         OString `json:"-"` // internal
	ResolvedShortName    OString `json:"-"` // internal
	ResolvedPlaces       OString `json:"-"` // internal
	OperatorID           OInt    `json:"-"` // internal
	ID                   int     `json:"-"` // internal
	FeedID               int     `json:"-"` // internal
}

func (o OperatorAssociatedFeed) TableName() string {
	return "current_operators_in_feed"
}

// OperatorAssociatedFeeds is necessary to scan correctly from database
type OperatorAssociatedFeeds []OperatorAssociatedFeed

// Value .
func (a OperatorAssociatedFeeds) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan .
func (a *OperatorAssociatedFeeds) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
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

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (r *OperatorAssociatedFeeds) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

// MarshalGQL implements the graphql.Marshaler interface
func (r OperatorAssociatedFeeds) MarshalGQL(w io.Writer) {
	b, _ := json.Marshal(&r)
	w.Write(b)
}
