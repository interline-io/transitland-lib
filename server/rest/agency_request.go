package rest

const agencyQuery = `
query ($limit: Int, $ids: [Int!], $where:AgencyFilter) {
	agencies(limit: $limit, ids: $ids, where: $where) {
	  id
	  agency_name
	  agency_id
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
	  routes(limit:1000) {
		id
		route_id
		route_short_name
		route_long_name
	  }	  
	}
  }
`

// AgencyRequest holds options for a Route request
type AgencyRequest struct {
	ID              int     `json:"id,string"`
	Limit           int     `json:"limit,string"`
	After           int     `json:"after,string"`
	AgencyID        string  `json:"agency_id"`
	AgencyName      string  `json:"agency_name"`
	OnestopID       string  `json:"onestop_id"`
	FeedVersionSHA1 string  `json:"feed_version_sha1"`
	FeedOnestopID   string  `json:"feed_onestop_id"`
	Lat             float64 `json:"lat,string"`
	Lon             float64 `json:"lon,string"`
	Radius          float64 `json:"radius,string"`
}

// ResponseKey returns the GraphQL response entity key.
func (r AgencyRequest) ResponseKey() string { return "agencies" }

// Query returns a GraphQL query string and variables.
func (r AgencyRequest) Query() (string, map[string]interface{}) {
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
	if r.Lat != 0.0 && r.Lon != 0.0 {
		where["near"] = hw{"lat": r.Lat, "lon": r.Lon, "radius": r.Radius}
	}
	return agencyQuery, hw{"limit": checkLimit(r.Limit), "after": checkAfter(r.After), "ids": checkIds(r.ID), "where": where}
}
