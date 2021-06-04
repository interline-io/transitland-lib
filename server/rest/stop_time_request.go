package rest

const stopTimeQuery = `
query ($limit: Int, $include_trip: Boolean!, $include_stop: Boolean!, $where: StopTimeFilter) {
	stop_times(limit: $limit, where: $where) {
	  arrival_time
	  departure_time
	  stop_sequence
	  stop_headsign
	  pickup_type
	  drop_off_type
	  timepoint
	  interpolated
	  trip_id
	  stop_id
	  trip @include(if: $include_trip) {
		id
		trip_id
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
	  stop @include(if: $include_stop) {
		id
		stop_id
		stop_name
		geometry
	  }
	}
  }
  
`

// StopTimeRequest holds options for a stop_times request
type StopTimeRequest struct {
	ID     int `json:"id,string"`
	Limit  int `json:"limit,string"`
	TripID int `json:"trip_id,string"`
	StopID int `json:"stop_id,string"`
}

// Query returns a GraphQL query string and variables.
func (r StopTimeRequest) Query() (string, map[string]interface{}) {
	r.Limit = checkLimit(r.Limit)
	where := hw{}
	includeTrip := false
	includeStop := false
	if r.StopID > 0 {
		where["stop_id"] = hw{"_eq": r.StopID}
		includeTrip = true
	}
	if r.TripID > 0 {
		where["trip_id"] = hw{"_eq": r.TripID}
	}
	return stopTimeQuery, hw{"limit": checkLimit(r.Limit), "where": where, "include_trip": includeTrip, "include_stop": includeStop}
}
