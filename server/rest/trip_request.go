package rest

import (
	"context"
	_ "embed"
	"strconv"
	"strings"

	oa "github.com/getkin/kin-openapi/openapi3"
)

//go:embed trip_request.gql
var tripQuery string

// TripRequest holds options for a /trips request
type TripRequest struct {
	ID               int    `json:"id,string"`
	TripID           string `json:"trip_id"`
	RouteKey         string `json:"route_key"`
	RouteID          int    `json:"route_id,string"`
	RouteOnestopID   string `json:"route_onestop_id"`
	FeedOnestopID    string `json:"feed_onestop_id"`
	FeedVersionSHA1  string `json:"feed_version_sha1"`
	ServiceDate      string `json:"service_date"`
	RelativeDate     string `json:"relative_date"`
	IncludeGeometry  bool   `json:"include_geometry,string"`
	IncludeStopTimes bool   `json:"include_stop_times,string"`
	IncludeAlerts    bool   `json:"include_alerts,string"`
	Format           string
	LicenseFilter
	WithCursor
}

func (r TripRequest) RequestInfo() RequestInfo {
	return RequestInfo{
		Path: "/routes/{route_key}/trips",
		Get: &RequestOperation{
			Query: tripQuery,
			Operation: &oa.Operation{
				Summary: `Search for trips`,
				Extensions: map[string]any{
					"x-alternates": []RequestAltPath{
						{"GET", "/routes/{route_key}/trips.{format}", "Request trips in specified format"},
						{"GET", "/routes/{route_key}/trips/{id}", "Request a trip by ID"},
						{"GET", "/routes/{route_key}/trips/{id}.format", "Request a trip by ID in specified format"},
					},
				},
				Parameters: oa.Parameters{
					&pref{Value: &param{
						Name:        "route_key",
						In:          "path",
						Description: `Route lookup key; can be an integer ID, a '<feed onestop_id>:<gtfs route_id>' key, or a Onestop ID`,
						Required:    true,
						Schema:      newSRVal("string", "", nil),
					}},
					&pref{Value: &param{
						Name:        "service_date",
						In:          "query",
						Description: `Search for trips active on this date`,
						Schema:      newSRVal("string", "date", nil),
						Extensions:  newExt("", "service_date=...", "route_onestop_id=r-9q9j-l1&service_date=2021-07-14"),
					}},
					&pref{Value: &param{
						Name:        "trip_id",
						In:          "query",
						Description: `Search for records with this GTFS trip_id`,
						Schema:      newSRVal("string", "", nil),
						Extensions:  newExt("", "trip_id=305", "route_onestop_id=r-9q9j-l1&trip_id=305"),
					}},
					&pref{Value: &param{
						Name:        "include_geometry",
						In:          "query",
						Description: `Include shape geometry`,
						Schema:      newSRVal("string", "", []any{"true", "false"}),
						Extensions:  newExt("", "include_geometry=true", "route_onestop_id=r-9q9j-l1&include_geometry=true"),
					}},
					&pref{Value: &param{
						Name:        "use_service_window",
						In:          "query",
						Description: `Use a fall-back service date if the requested service_date is outside the active service period of the feed version. The fall-back date is selected as the matching day-of-week in the week which provides the best level of scheduled service in the feed version. This value defaults to true.`,
						Schema:      newSRVal("string", "", []any{"true", "false"}),
						Extensions:  newExt("", "use_service_window=false", "route_onestop_id=r-9q9j-l1&use_service_window=false"),
					}},
					newPRefExt("relativeDateParam", "", "relative_date=NEXT_MONDAY", "route_onestop_id=r-9q9j-l1&relative_date=NEXT_MONDAY"),
					newPRef("includeAlertsParam"),
					newPRef("idParam"),
					newPRef("afterParam"),
					newPRefExt("limitParam", "", "limit=1", "route_onestop_id=r-9q9j-l1&limit=10&limit=1"),
					newPRefExt("formatParam", "", "format=geojson", "route_onestop_id=r-9q9j-l1&limit=10&format=geojson"),
					newPRefExt("sha1Param", "", "feed_version_sha1=041ffeec...", "route_onestop_id=r-9q9j-l1&feed_version_sha1=041ffeec98316e560bc2b91960f7150ad329bd5f"),
					newPRefExt("feedParam", "", "feed_onestop_id=f-sf~bay~area~rg", "route_onestop_id=r-9q9j-l1&feed_onestop_id=f-sf~bay~area~rg"),
					newPRef("latParam"),
					newPRef("lonParam"),
					newPRef("licenseCommercialUseAllowedParam"),
					newPRef("licenseShareAlikeOptionalParam"),
					newPRef("licenseCreateDerivedProductParam"),
					newPRef("licenseRedistributionAllowedParam"),
					newPRef("licenseUseWithoutAttributionParam"),
				},
			},
		},
	}
}

// ResponseKey .
func (r TripRequest) ResponseKey() string {
	return "trips"
}

// Query returns a GraphQL query string and variables.
func (r TripRequest) Query(ctx context.Context) (string, map[string]interface{}) {
	// ID or RouteID should be considered mandatory.
	if r.RouteKey == "" {
		// pass
	} else if v, err := strconv.Atoi(r.RouteKey); err == nil {
		r.RouteID = v
	} else {
		r.RouteOnestopID = r.RouteKey
	}
	where := hw{}
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
	if r.RelativeDate != "" {
		where["relative_date"] = strings.ToUpper(r.RelativeDate)
	} else if r.ServiceDate != "" {
		where["service_date"] = r.ServiceDate
	}
	where["license"] = checkLicenseFilter(r.LicenseFilter)
	// Include geometry when in geojson format
	if r.ID > 0 || r.Format == "geojson" || r.Format == "geojsonl" {
		r.IncludeGeometry = true
	}
	// Only include stop times when requesting a specific trip.
	r.IncludeStopTimes = false
	if r.ID > 0 {
		r.IncludeStopTimes = true
	}
	includeRoute := false
	return tripQuery, hw{
		"limit":              r.CheckLimit(),
		"after":              r.CheckAfter(),
		"ids":                checkIds(r.ID),
		"where":              where,
		"include_geometry":   r.IncludeGeometry,
		"include_stop_times": r.IncludeStopTimes,
		"include_route":      includeRoute,
		"include_alerts":     r.IncludeAlerts,
	}
}

// ProcessGeoJSON .
func (r TripRequest) ProcessGeoJSON(ctx context.Context, response map[string]interface{}) error {
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
	return processGeoJSON(ctx, r, response)
}

///////////

type TripEntityRequest struct {
	TripRequest
}

func (r TripEntityRequest) RequestInfo() RequestInfo {
	return RequestInfo{
		Path: "/routes/{route_key}/trips/{id}",
		Get: &RequestOperation{
			Query: tripQuery,
			Operation: &oa.Operation{
				Summary: `Search for trips`,
				Parameters: oa.Parameters{
					&pref{Value: &param{
						Name:        "route_key",
						In:          "path",
						Description: `Route lookup key; can be an integer ID, a '<feed onestop_id>:<gtfs route_id>' key, or a Onestop ID`,
						Required:    true,
						Schema:      newSRVal("string", "", nil),
					}},
					&pref{Value: &param{
						Name:        "id",
						In:          "path",
						Required:    true,
						Description: `Trip ID`,
						Schema:      newSRVal("integer", "", nil),
					}},
					&pref{Value: &param{
						Name:        "include_geometry",
						In:          "query",
						Description: `Include shape geometry`,
						Schema:      newSRVal("string", "", []any{"true", "false"}),
						Extensions:  newExt("", "include_geometry=true", "route_onestop_id=r-9q9j-l1&include_geometry=true"),
					}},
					newPRef("includeAlertsParam"),
					newPRefExt("limitParam", "", "limit=1", "route_onestop_id=r-9q9j-l1&limit=1"),
					newPRefExt("formatParam", "", "format=geojson", "route_onestop_id=r-9q9j-l1&format=geojson"),
				},
			},
		},
	}
}
