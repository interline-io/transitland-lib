package rest

const operatorQuery = `
query ($limit: Int, $after: Int, $where:OperatorFilter) {
	operators(after: $after, limit: $limit, where: $where) {
	  id
	  agency_name
	  operator_name
	  operator_short_name
	  onestop_id
	  city_name
	  adm1name
	  adm0name
	  places_cache
	  agency {
		places(where:{min_rank:0.2}) {
		  name
		  adm0name
		  adm1name
		}
	  }
	}
  }
`

// OperatorRequest holds options for a Route request
type OperatorRequest struct {
	ID              int    `json:"id,string"`
	Limit           int    `json:"limit,string"`
	After           int    `json:"after,string"`
	OnestopID       string `json:"onestop_id"`
	FeedVersionSHA1 string `json:"feed_version_sha1"`
	FeedOnestopID   string `json:"feed_onestop_id"`
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
	// if r.Lat != 0.0 && r.Lon != 0.0 {
	// 	where["near"] = hw{"lat": r.Lat, "lon": r.Lon, "radius": r.Radius}
	// }
	return operatorQuery, hw{"limit": checkLimit(r.Limit), "after": checkAfter(r.After), "ids": checkIds(r.ID), "where": where}
}
