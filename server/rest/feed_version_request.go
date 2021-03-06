package rest

import (
	"strconv"
)

const feedVersionQuery = `
query ($limit: Int, $ids:[Int!], $where:FeedVersionFilter ) {
	feed_versions(limit: $limit, ids: $ids, where: $where) {
	  id
	  sha1
	  fetched_at
	  url
	  earliest_calendar_date
	  latest_calendar_date
	  files {
		id
		name
		rows
		sha1
		header
		csv_like
		size
	  }
      service_levels {
        start_date
        end_date
        monday
        tuesday
        wednesday
        thursday
        friday
        saturday
        sunday
        route_id
      }
	  feed_version_gtfs_import {
	    id
	    in_progress
	    success
	    exception_log
	  }
	}
  }  
`

// FeedVersionRequest holds options for a Route request
type FeedVersionRequest struct {
	Key           string `json:"key"`
	ID            int    `json:"id,string"`
	Limit         int    `json:"limit"`
	After         int    `json:"after"`
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
