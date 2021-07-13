package rest

const tripQuery = `
query(
	$limit: Int
	$ids: [Int!]
	$include_route: Boolean!
	$include_stop_times: Boolean!
	$include_geometry: Boolean!
	$where: TripFilter
  ) {
	trips(limit: $limit, ids: $ids, where: $where) {
	  id
	  trip_id
	  trip_headsign
	  trip_short_name
	  direction_id
	  block_id
	  wheelchair_accessible
	  bikes_allowed
	  stop_pattern_id
	  feed_version {
		sha1
		fetched_at
		feed {
		  id
		  onestop_id
		}
	  }
	  shape {
		shape_id
		geometry @include(if: $include_geometry)
		generated
	  }
	  calendar {
		service_id
		start_date
		end_date
		monday
		tuesday
		wednesday
		thursday
		friday
		saturday
		sunday
		added_dates
		removed_dates
	  }
	  frequencies {
		start_time
		end_time
		headway_secs
		exact_times
	  }
	  route @include(if: $include_route) {
		id
		onestop_id
		route_id
		route_short_name
		route_long_name
		agency {
		  id
		  agency_id
		  agency_name
		}
	  }
	  stop_times @include(if: $include_stop_times) {
		arrival_time
		departure_time
		stop_sequence
		stop_headsign
		pickup_type
		drop_off_type
		timepoint
		interpolated
		stop {
		  id
		  stop_id
		  stop_name
		  geometry
		}
	  }
	}
  }	
`

// TripRequest holds options for a /trips request
type TripRequest struct {
	ID               int    `json:"id,string"`
	Limit            int    `json:"limit,string"`
	After            int    `json:"after,string"`
	RouteID          int    `json:"route_id,string"`
	TripID           string `json:"trip_id,string"`
	FeedOnestopID    string `json:"feed_onestop_id,string"`
	FeedVersionSHA1  string `json:"feed_version_sha1"`
	IncludeGeometry  string `json:"include_geometry"`
	IncludeStopTimes string `json:"include_stop_times"`
	ServiceDate      string `json:"service_date"`
	Format           string
}

// ResponseKey .
func (r TripRequest) ResponseKey() string {
	return "trips"
}

// Query returns a GraphQL query string and variables.
func (r TripRequest) Query() (string, map[string]interface{}) {
	// ID or RouteID should be considered mandatory.
	where := hw{}
	if r.RouteID > 0 {
		where["route_id"] = r.RouteID
	}
	if r.FeedOnestopID != "" {
		where["feed_onestop_id"] = r.FeedOnestopID
	}
	if r.FeedVersionSHA1 != "" {
		where["feed_version_sha1"] = r.FeedVersionSHA1
	}
	if r.TripID != "" {
		where["trip_id"] = r.TripID
	}
	if r.ServiceDate != "" {
		where["service_date"] = r.ServiceDate
	}
	// Include geometry when in geojson format
	includeGeometry := false
	if r.ID > 0 || r.IncludeGeometry == "true" || r.Format == "geojson" {
		includeGeometry = true
	}
	// Only include stop times when requesting a specific trip.
	includeStopTimes := false
	if r.ID > 0 || r.IncludeStopTimes == "true" || r.Format == "geojson" {
		includeStopTimes = true
	}
	includeRoute := false
	return tripQuery, hw{"limit": checkLimit(r.Limit), "after": checkAfter(r.After), "ids": checkIds(r.ID), "where": where, "include_geometry": includeGeometry, "include_stop_times": includeStopTimes, "include_route": includeRoute}
}

// ProcessGeoJSON .
func (r TripRequest) ProcessGeoJSON(response map[string]interface{}) error {
	entities, ok := response[r.ResponseKey()].([]interface{})
	if ok {
		for _, feature := range entities {
			if f2, ok := feature.(map[string]interface{}); ok {
				shp := feature.(map[string]interface{})["shape"].(map[string]interface{})
				f2["geometry"] = shp["geometry"]
				delete(shp, "geometry")
			}
		}
	}
	return processGeoJSON(r, response)
}
