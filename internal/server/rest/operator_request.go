package rest

import (
	_ "embed"
)

//go:embed operator_request.gql
var operatorQuery string

// OperatorRequest holds options for a Route request
type OperatorRequest struct {
	ID              int    `json:"id,string"`
	Limit           int    `json:"limit,string"`
	After           int    `json:"after,string"`
	OnestopID       string `json:"onestop_id"`
	FeedVersionSHA1 string `json:"feed_version_sha1"`
	FeedOnestopID   string `json:"feed_onestop_id"`
	Search          string `json:"search"`
	// Lat             float64 `json:"lat,string"`
	// Lon             float64 `json:"lon,string"`
	// Radius          float64 `json:"radius,string"`
}

// ResponseKey returns the GraphQL response entity key.
func (r OperatorRequest) ResponseKey() string { return "operators" }

// Query returns a GraphQL query string and variables.
func (r OperatorRequest) Query() (string, map[string]interface{}) {
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
	if r.Search != "" {
		where["search"] = r.Search
	}
	// if r.Lat != 0.0 && r.Lon != 0.0 {
	// 	where["near"] = hw{"lat": r.Lat, "lon": r.Lon, "radius": r.Radius}
	// }
	return operatorQuery, hw{"limit": checkLimit(r.Limit), "after": checkAfter(r.After), "ids": checkIds(r.ID), "where": where}
}
