package rest

import (
	_ "embed"
	"strconv"
)

//go:embed feed_version_request.gql
var feedVersionQuery string

// FeedVersionRequest holds options for a Route request
type FeedVersionRequest struct {
	FeedVersionKey string `json:"feed_version_key"`
	FeedKey        string `json:"feed_key"`
	ID             int    `json:"id,string"`
	Limit          int    `json:"limit,string"`
	After          int    `json:"after,string"`
	FeedID         int    `json:"feed_id,string"`
	FeedOnestopID  string `json:"feed_onestop_id"`
	Sha1           string `json:"sha1"`
}

// Query returns a GraphQL query string and variables.
func (r FeedVersionRequest) Query() (string, map[string]interface{}) {
	// Handle feed key
	if r.FeedKey == "" {
		// pass
	} else if v, err := strconv.Atoi(r.FeedKey); err == nil {
		r.FeedID = v
	} else {
		r.FeedOnestopID = r.FeedKey
	}
	// Handle feed version key
	if r.FeedVersionKey == "" {
		// pass
	} else if v, err := strconv.Atoi(r.FeedVersionKey); err == nil {
		r.ID = v
	} else {
		r.Sha1 = r.FeedVersionKey
	}
	where := hw{}
	if r.FeedID > 0 {
		where["feed_ids"] = []int{r.FeedID}
	}
	if r.FeedOnestopID != "" {
		where["feed_onestop_id"] = r.FeedOnestopID
	}
	if r.Sha1 != "" {
		where["sha1"] = r.Sha1
	}
	return feedVersionQuery, hw{"limit": checkLimit(r.Limit), "after": checkAfter(r.After), "ids": checkIds(r.ID), "where": where}
}

// ResponseKey .
func (r FeedVersionRequest) ResponseKey() string {
	return "feed_versions"
}
