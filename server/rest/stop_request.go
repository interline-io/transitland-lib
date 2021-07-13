package rest

import "strconv"

const stopQuery = `
query($limit: Int, $ids: [Int!], $where: StopFilter) {
	stops(limit: $limit, ids: $ids, where: $where) {
	  id
	  stop_id
	  stop_name
	  stop_url
	  stop_timezone
	  stop_desc
	  stop_code
	  zone_id
	  wheelchair_boarding
	  location_type
	  onestop_id
	  geometry
	  feed_version {
		id
		sha1
		fetched_at
		feed {
		  id
		  onestop_id
		}
	  }
	  level {
		level_id
		level_name
		level_index
	  }
	  parent {
		id
		stop_id
		stop_name
		geometry
	  }
	  route_stops(limit: 1000) {
		route {
		  id
		  route_id
		  route_short_name
		  route_long_name
		  agency {
			id
			agency_id
			agency_name
		  }
		}
	  }
	}
  }
  
`

// StopRequest holds options for a /stops request
type StopRequest struct {
	Key             string  `json:"key"`
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
