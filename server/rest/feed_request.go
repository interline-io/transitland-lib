package rest

import (
	_ "embed"
	"strconv"
)

//go:embed feed_request.gql
var feedQuery string

// FeedRequest holds options for a Route request
type FeedRequest struct {
	Key        string `json:"key"`
	ID         int    `json:"id,string"`
	Limit      int    `json:"limit,string"`
	After      int    `json:"after,string"`
	OnestopID  string `json:"onestop_id"`
	Spec       string `json:"spec"`
	Search     string `json:"search"`
	FetchError string `json:"fetch_error"`
	// Lat       float64 `json:"lat,string"`
	// Lon       float64 `json:"lon,string"`
	// Radius    float64 `json:"radius,string"`
}

// Query returns a GraphQL query string and variables.
func (r FeedRequest) Query() (string, map[string]interface{}) {
	if r.Key == "" {
		// pass
	} else if v, err := strconv.Atoi(r.Key); err == nil {
		r.ID = v
	} else {
		r.OnestopID = r.Key
	}
	where := hw{}
	if r.OnestopID != "" {
		where["onestop_id"] = r.OnestopID
	}
	if r.Spec != "" {
		where["spec"] = []string{r.Spec}
	}
	if r.Search != "" {
		where["search"] = r.Search
	}
	if r.FetchError == "true" {
		where["fetch_error"] = true
	} else if r.FetchError == "false" {
		where["fetch_error"] = false
	}
	return feedQuery, hw{"limit": checkLimit(r.Limit), "after": checkAfter(r.After), "ids": checkIds(r.ID), "where": where}
}
