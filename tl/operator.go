package tl

type Operator struct {
	ID              int                      `json:"-"`
	OnestopID       OString                  `json:"onestop_id"`
	Tags            Tags                     `json:"tags" db:"operator_tags"`
	Name            OString                  `json:"name"`
	ShortName       OString                  `json:"short_name"`
	Website         OString                  `json:"website"`
	AssociatedFeeds []OperatorAssociatedFeed `json:"associated_feeds" db:"-"`
	DeletedAt       OTime                    `json:"-"`
	Timestamps
}

type OperatorAssociatedFeed struct {
	FeedOnestopID OString `json:"feed_onestop_id"`
	AgencyID      OString `json:"gtfs_agency_id" db:"gtfs_agency_id"`
	ID            int     `json:"-"` // internal
	FeedID        int     `json:"-"` // internal
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
