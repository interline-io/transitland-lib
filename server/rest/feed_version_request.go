package rest

import (
	_ "embed"
	"strconv"
)

//go:embed feed_version_request.gql
var feedVersionQuery string

// FeedVersionRequest holds options for a Route request
type FeedVersionRequest struct {
	Key           string `json:"key"`
	ID            int    `json:"id,string"`
	Limit         int    `json:"limit,string"`
	After         int    `json:"after,string"`
	FeedOnestopID string `json:"feed_onestop_id"`
	Sha1          string `json:"sha1"`
}

// Query returns a GraphQL query string and variables.
func (r FeedVersionRequest) Query() (string, map[string]interface{}) {
	if r.Key == "" {
		// pass
	} else if v, err := strconv.Atoi(r.Key); err == nil {
		r.ID = v
	} else {
		r.Sha1 = r.Key
	}
	where := hw{}
	if r.FeedOnestopID != "" {
		where["feed_onestop_id"] = r.FeedOnestopID
	}
	if r.Sha1 != "" {
		where["sha1"] = r.Sha1
	}
	return feedVersionQuery, hw{"limit": checkLimit(r.Limit), "after": checkAfter(r.After), "ids": checkIds(r.ID), "where": where}
}
