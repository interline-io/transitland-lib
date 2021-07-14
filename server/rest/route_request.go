package rest

import (
	_ "embed"
	"strconv"
)

//go:embed route_request.gql
var routeQuery string

// RouteRequest holds options for a Route request
type RouteRequest struct {
	Key               string  `json:"key"`
	ID                int     `json:"id,string"`
	Limit             int     `json:"limit,string"`
	After             int     `json:"after,string"`
	RouteID           string  `json:"route_id"`
	RouteType         string  `json:"route_type"`
	OnestopID         string  `json:"onestop_id"`
	OperatorOnestopID string  `json:"operator_onestop_id"`
	IncludeGeometry   string  `json:"include_geometry"`
	Format            string  `json:"format"`
	Search            string  `json:"search"`
	AgencyID          int     `json:"agency_id,string"`
	FeedVersionSHA1   string  `json:"feed_version_sha1"`
	FeedOnestopID     string  `json:"feed_onestop_id"`
	Lat               float64 `json:"lat,string"`
	Lon               float64 `json:"lon,string"`
	Radius            float64 `json:"radius,string"`
}

// ResponseKey returns the GraphQL response entity key.
func (r RouteRequest) ResponseKey() string { return "routes" }

// Query returns a GraphQL query string and variables.
func (r RouteRequest) Query() (string, map[string]interface{}) {
	if r.Key == "" {
		// pass
	} else if v, err := strconv.Atoi(r.Key); err == nil {
		r.ID = v
	} else {
		r.OnestopID = r.Key
	}
	where := hw{}
	if r.FeedVersionSHA1 != "" {
		where["feed_version_sha1"] = r.FeedVersionSHA1
	}
	if r.FeedOnestopID != "" {
		where["feed_onestop_id"] = r.FeedOnestopID
	}
	if r.RouteID != "" {
		where["route_id"] = r.RouteID
	}
	if r.RouteType != "" {
		where["route_type"] = r.RouteType
	}
	if r.OnestopID != "" {
		where["onestop_id"] = r.OnestopID
	}
	if r.OperatorOnestopID != "" {
		where["operator_onestop_id"] = r.OperatorOnestopID
	}
	if r.AgencyID > 0 {
		where["agency_id"] = r.AgencyID
	}
	if r.Lat != 0.0 && r.Lon != 0.0 {
		where["near"] = hw{"lat": r.Lat, "lon": r.Lon, "radius": r.Radius}
	}
	if r.Search != "" {
		where["search"] = r.Search
	}
	includeGeometry := false
	if r.IncludeGeometry == "true" || r.Format == "geojson" || r.Format == "png" {
		includeGeometry = true
	}
	return routeQuery, hw{"limit": checkLimit(r.Limit), "after": checkAfter(r.After), "ids": checkIds(r.ID), "where": where, "include_geometry": includeGeometry}
}
