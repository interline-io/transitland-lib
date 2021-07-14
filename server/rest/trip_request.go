package rest

import (
	_ "embed"
	"strconv"
)

//go:embed trip_request.gql
var tripQuery string

// TripRequest holds options for a /trips request
type TripRequest struct {
	ID               int    `json:"id,string"`
	Limit            int    `json:"limit,string"`
	After            int    `json:"after,string"`
	TripID           string `json:"trip_id,string"`
	RouteKey         string `json:"route_key"`
	RouteID          int    `json:"route_id,string"`
	RouteOnestopID   string `json:"route_onestop_id,string"`
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
	if r.RouteKey == "" {
		// pass
	} else if v, err := strconv.Atoi(r.RouteKey); err == nil {
		r.RouteID = v
	} else {
		r.RouteOnestopID = r.RouteKey
	}
	if r.RouteID > 0 {
		where["route_ids"] = []int{r.RouteID}
	}
	if r.RouteOnestopID != "" {
		where["route_onestop_ids"] = []string{r.RouteOnestopID}
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
