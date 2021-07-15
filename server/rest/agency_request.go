package rest

import (
	_ "embed"
	"strconv"
)

//go:embed agency_request.gql
var agencyQuery string

// AgencyRequest holds options for a Route request
type AgencyRequest struct {
	ID              int     `json:"id,string"`
	Limit           int     `json:"limit,string"`
	After           int     `json:"after,string"`
	AgencyKey       string  `json:"agency_key"`
	AgencyID        string  `json:"agency_id"`
	AgencyName      string  `json:"agency_name"`
	OnestopID       string  `json:"onestop_id"`
	FeedVersionSHA1 string  `json:"feed_version_sha1"`
	FeedOnestopID   string  `json:"feed_onestop_id"`
	Search          string  `json:"search"`
	Lat             float64 `json:"lat,string"`
	Lon             float64 `json:"lon,string"`
	Radius          float64 `json:"radius,string"`
}

// ResponseKey returns the GraphQL response entity key.
func (r AgencyRequest) ResponseKey() string { return "agencies" }

// Query returns a GraphQL query string and variables.
func (r AgencyRequest) Query() (string, map[string]interface{}) {
	if r.AgencyKey == "" {
		// pass
	} else if v, err := strconv.Atoi(r.AgencyKey); err == nil {
		r.ID = v
	} else {
		r.OnestopID = r.AgencyKey
	}
	where := hw{}
	if r.FeedVersionSHA1 != "" {
		where["feed_version_sha1"] = r.FeedVersionSHA1
	}
	if r.FeedOnestopID != "" {
		where["feed_onestop_id"] = r.FeedOnestopID
	}
	if r.OnestopID != "" {
		where["onestop_id"] = r.OnestopID
	}
	if r.AgencyID != "" {
		where["agency_id"] = r.AgencyID
	}
	if r.AgencyName != "" {
		where["agency_name"] = r.AgencyName
	}
	if r.Search != "" {
		where["search"] = r.Search
	}
	if r.Lat != 0.0 && r.Lon != 0.0 {
		where["near"] = hw{"lat": r.Lat, "lon": r.Lon, "radius": r.Radius}
	}
	return agencyQuery, hw{"limit": checkLimit(r.Limit), "after": checkAfter(r.After), "ids": checkIds(r.ID), "where": where}
}
