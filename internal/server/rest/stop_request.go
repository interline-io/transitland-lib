package rest

import (
	_ "embed"
	"strconv"
)

//go:embed stop_request.gql
var stopQuery string

// StopRequest holds options for a /stops request
type StopRequest struct {
	StopKey         string  `json:"stop_key"`
	ID              int     `json:"id,string"`
	Limit           int     `json:"limit,string"`
	After           int     `json:"after,string"`
	StopID          string  `json:"stop_id"`
	OnestopID       string  `json:"onestop_id"`
	FeedVersionSHA1 string  `json:"feed_version_sha1"`
	FeedOnestopID   string  `json:"feed_onestop_id"`
	Search          string  `json:"search"`
	Lat             float64 `json:"lat,string"`
	Lon             float64 `json:"lon,string"`
	Radius          float64 `json:"radius,string"`
}

// ResponseKey returns the GraphQL response entity key.
func (r StopRequest) ResponseKey() string { return "stops" }

// Query returns a GraphQL query string and variables.
func (r StopRequest) Query() (string, map[string]interface{}) {
	if r.StopKey == "" {
		// pass
	} else if v, err := strconv.Atoi(r.StopKey); err == nil {
		r.ID = v
	} else {
		r.OnestopID = r.StopKey
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
	if r.StopID != "" {
		where["stop_id"] = r.StopID
	}
	if r.Lat != 0.0 && r.Lon != 0.0 {
		where["near"] = hw{"lat": r.Lat, "lon": r.Lon, "radius": r.Radius}
	}
	if r.Search != "" {
		where["search"] = r.Search
	}
	return stopQuery, hw{"limit": checkLimit(r.Limit), "after": checkAfter(r.After), "ids": checkIds(r.ID), "where": where}
}
