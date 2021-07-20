package tl

type Operator struct {
	OnestopID       OString                  `json:"onestop_id"`
	Tags            Tags                     `json:"tags" db:"feed_tags"`
	Name            OString                  `json:"name"`
	ShortName       OString                  `json:"short_name"`
	Website         OString                  `json:"website"`
	AssociatedFeeds []OperatorAssociatedFeed `json:"associated_feeds" db:"-"`
}

type OperatorAssociatedFeed struct {
	FeedOnestopID OString `json:"feed_onestop_id"`
	AgencyID      OString `json:"gtfs_agency_id" db:"gtfs_agency_id"`
}
