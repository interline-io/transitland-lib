package rest

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"

	oa "github.com/getkin/kin-openapi/openapi3"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
)

//go:embed route_pattern_request.gql
var routePatternQuery string

// RoutePatternRequest holds options for a /routes/{route_key}/patterns request
type RoutePatternRequest struct {
	RouteKey         string `json:"route_key"`
	ServiceDate      string `json:"service_date"`
	RelativeDate     string `json:"relative_date"`
	UseServiceWindow *bool  `json:"use_service_window,string"`
	IncludeTimetable bool   `json:"include_timetable,string"`
	IncludeShapes    bool   `json:"include_shapes,string"`
	IncludeStops     bool   `json:"include_stops,string"`
	Encoding         string `json:"encoding"`
	LicenseFilter
}

// RequestInfo describes the endpoint and its parameters.
func (r RoutePatternRequest) RequestInfo() RequestInfo {
	return RequestInfo{
		Path: "/routes/{route_key}/patterns",
		Get: &RequestOperation{
			Query: routePatternQuery,
			Operation: &oa.Operation{
				Summary: `Stop patterns for a route, with their timetables and shapes`,
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
						Description: `Return only the patterns that trips run on this date, counting their trips and building their timetables from this date's service. Required unless relative_date is given.`,
						Schema:      newSRVal("string", "date", nil),
					}},
					newPRef("relativeDateParam"),
					&pref{Value: &param{
						Name:        "use_service_window",
						In:          "query",
						Description: `Use a fall-back service date if the requested date is outside the active service period of the feed version: the matching day of the week in the week with the most scheduled service. Defaults to true.`,
						Schema:      newSRVal("string", "", []any{"true", "false"}),
					}},
					&pref{Value: &param{
						Name:        "include_timetable",
						In:          "query",
						Description: `Include each pattern's timetable: its trips, and grids of their arrival times, departure times and distances along their shapes. Needs a date.`,
						Schema:      newSRVal("string", "", []any{"true", "false"}),
					}},
					&pref{Value: &param{
						Name:        "include_shapes",
						In:          "query",
						Description: `Include the shapes each pattern's trips follow, each once, with the distance along it at each point`,
						Schema:      newSRVal("string", "", []any{"true", "false"}),
					}},
					&pref{Value: &param{
						Name:        "include_stops",
						In:          "query",
						Description: `Include each pattern's stops, in order`,
						Schema:      newSRVal("string", "", []any{"true", "false"}),
					}},
					&pref{Value: &param{
						Name:        "encoding",
						In:          "query",
						Description: `How timetable grids and shape geometries are written: plain arrays, or encoded polylines. polyline6 encodes shape geometries at six digits of precision rather than five; timetable grids hold whole numbers either way. Defaults to array.`,
						Schema:      newSRVal("string", "", []any{"array", "polyline", "polyline6"}),
					}},
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

// Validate requires a route key, a date in a form the API reads, and an encoding
// the endpoint knows.
func (r RoutePatternRequest) Validate() error {
	if r.RouteKey == "" {
		return errors.New("route_key is required")
	}
	if r.RelativeDate != "" {
		if !model.RelativeDate(strings.ToUpper(r.RelativeDate)).IsValid() {
			return fmt.Errorf("relative_date %q is not a relative date", r.RelativeDate)
		}
	} else if r.ServiceDate != "" {
		if _, err := tt.ParseDate(r.ServiceDate); err != nil {
			return fmt.Errorf("service_date %q is not a date in YYYY-MM-DD form", r.ServiceDate)
		}
	} else {
		return errors.New("service_date or relative_date is required")
	}
	switch r.Encoding {
	case "", "array", "polyline", "polyline6":
		return nil
	}
	return fmt.Errorf("encoding must be array, polyline or polyline6, not %q", r.Encoding)
}

// ResponseKey returns the GraphQL response entity key.
func (r RoutePatternRequest) ResponseKey() string { return "routes" }

// IncludeNext is false: the patterns are not paged. A route Onestop ID can still
// match a route in more than one feed's active version.
func (r RoutePatternRequest) IncludeNext() bool { return false }

// Query returns a GraphQL query string and variables.
func (r RoutePatternRequest) Query(ctx context.Context) (string, map[string]interface{}) {
	k := parseRouteKey(r.RouteKey)
	where := hw{}
	if k.FeedOnestopID != "" {
		where["feed_onestop_id"] = k.FeedOnestopID
		where["route_id"] = k.RouteID
	} else if k.ID == 0 {
		where["onestop_id"] = k.OnestopID
	}
	where["license"] = checkLicenseFilter(r.LicenseFilter)
	patternWhere := hw{}
	if r.RelativeDate != "" {
		patternWhere["relative_date"] = strings.ToUpper(r.RelativeDate)
	} else if r.ServiceDate != "" {
		patternWhere["service_date"] = r.ServiceDate
	}
	if r.UseServiceWindow == nil || *r.UseServiceWindow {
		patternWhere["use_service_window"] = true
	}
	polyline := r.Encoding == "polyline"
	polyline6 := r.Encoding == "polyline6"
	return routePatternQuery, hw{
		"ids":               checkIds(k.ID),
		"where":             where,
		"pattern_where":     patternWhere,
		"include_timetable": r.IncludeTimetable,
		"include_shapes":    r.IncludeShapes,
		"include_stops":     r.IncludeStops,
		"array":             !polyline && !polyline6,
		"polyline":          polyline,
		"polyline6":         polyline6,
	}
}
