package dmfr

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"io"

	"github.com/interline-io/transitland-lib/tt"
)

type Operator struct {
	OnestopID         tt.String               `json:"onestop_id"`
	SupersedesIDs     tt.Strings              `json:"supersedes_ids,omitempty" db:"-"`
	Name              tt.String               `json:"name,omitempty"`
	ShortName         tt.String               `json:"short_name,omitempty"`
	Website           tt.String               `json:"website,omitempty"`
	AssociatedFeeds   OperatorAssociatedFeeds `json:"associated_feeds,omitempty"`
	Tags              tt.Tags                 `json:"tags,omitempty" db:"operator_tags"`
	File              tt.String               `json:"-"` // internal
	DeletedAt         tt.Time                 `json:"-"` // internal
	tt.DatabaseEntity `json:"-"`
	tt.Timestamps     `json:"-"`
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
	if ent.File != other.File {
		return false
	}
	return string(a1j) == string(a2j)
}

// TableName .
func (Operator) TableName() string {
	return "current_operators"
}

////////////

type OperatorAssociatedFeed struct {
	GtfsAgencyID         tt.String  `json:"gtfs_agency_id,omitempty" db:"gtfs_agency_id"`
	FeedOnestopID        tt.String  `json:"feed_onestop_id,omitempty" db:"-"`
	ResolvedOnestopID    tt.String  `json:"-"` // internal
	ResolvedGtfsAgencyID tt.String  `json:"-"` // internal
	ResolvedName         tt.String  `json:"-"` // internal
	ResolvedShortName    tt.String  `json:"-"` // internal
	ResolvedPlaces       tt.String  `json:"-"` // internal
	OperatorID           tt.Int     `json:"-"` // internal
	FeedID               int        `json:"-"` // internal
	tt.DatabaseEntity    `json:"-"` // internal
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
