package dmfr

import "github.com/interline-io/transitland-lib/tl/tt"

type FeedVersionAgencyOnestopID struct {
	EntityID  string
	OnestopID string
	tt.FeedVersionEntity
}

func (ent FeedVersionAgencyOnestopID) TableName() string {
	return "feed_version_agency_onestop_ids"
}

type FeedVersionRouteOnestopID struct {
	EntityID  string
	OnestopID string
	tt.FeedVersionEntity
}

func (ent FeedVersionRouteOnestopID) TableName() string {
	return "feed_version_route_onestop_ids"
}

type FeedVersionStopOnestopID struct {
	EntityID  string
	OnestopID string
	tt.FeedVersionEntity
}

func (ent FeedVersionStopOnestopID) TableName() string {
	return "feed_version_stop_onestop_ids"
}
