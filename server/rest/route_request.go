package rest

import (
	"context"
	_ "embed"
	"strconv"
	"strings"

	oa "github.com/getkin/kin-openapi/openapi3"
)

//go:embed route_request.gql
var routeQuery string

// RouteRequest holds options for a Route request
type RouteRequest struct {
	ID                int       `json:"id,string"`
	RouteKey          string    `json:"route_key"`
	AgencyKey         string    `json:"agency_key"`
	RouteID           string    `json:"route_id"`
	RouteType         string    `json:"route_type"`
	RouteTypes        string    `json:"route_types"`
	OnestopID         string    `json:"onestop_id"`
	OperatorOnestopID string    `json:"operator_onestop_id"`
	Format            string    `json:"format"`
	Search            string    `json:"search"`
	AgencyID          int       `json:"agency_id,string"`
	FeedVersionSHA1   string    `json:"feed_version_sha1"`
	FeedOnestopID     string    `json:"feed_onestop_id"`
	Lon               float64   `json:"lon,string"`
	Lat               float64   `json:"lat,string"`
	Radius            float64   `json:"radius,string"`
	Bbox              *restBbox `json:"bbox"`
	IncludeGeometry   bool      `json:"include_geometry,string"`
	IncludeAlerts     bool      `json:"include_alerts,string"`
	IncludeStops      bool      `json:"include_stops,string"`
	LicenseFilter
	WithCursor
}

func (r RouteRequest) RequestInfo() RequestInfo {
	return RequestInfo{
		Path: "/routes",
		Get: RequestOperation{
			Query: routeQuery,
			Operation: &oa.Operation{
				Summary: `Search for routes`,
				Extensions: map[string]any{
					"x-alternates": []RequestAltPath{
						{"GET", "/routes.{format}", "Request routes in specified format"},
						{"GET", "/routes/{route_key}", "Request a route by ID or Onestop ID"},
						{"GET", "/routes/{route_key}.format", "Request a route by ID or Onestop ID in specified format"},
					},
				},
				Parameters: oa.Parameters{
					&pref{Value: &param{
						Name:        "route_key",
						In:          "query",
						Description: `Route lookup key; can be an integer ID, a '<feed onestop_id>:<gtfs route_id>' key, or a Onestop ID`,
						Schema:      newSRVal("string", "", nil),
					}},
					&pref{Value: &param{
						Name:        "agency_key",
						In:          "query",
						Description: `Agency lookup key; can be an integer ID, a '<feed onestop_id>:<gtfs agency_id>' key, or a Onestop ID`,
						Schema:      newSRVal("string", "", nil),
					}},
					&pref{Value: &param{
						Name:        "route_id",
						In:          "query",
						Description: `Search for records with this GTFS route_id`,
						Schema:      newSRVal("string", "", nil),
						Extensions:  newExt("", "route_id=Bu-130", "feed_onestop_id=f-sf~bay~area~rg&route_id=AC:10"),
					}},
					&pref{Value: &param{
						Name:        "route_type",
						In:          "query",
						Description: `Search for routes with this GTFS route (vehicle) type`,
						Schema:      newSRVal("integer", "", nil),
						Extensions:  newExt("", "route_type=1", "route_type=1"),
					}},
					&pref{Value: &param{
						Name:        "route_types",
						In:          "query",
						Description: `Search for routes with these GTFS route (vehicle) types. Accepts comma separated values.`,
						Schema:      newSRVal("string", "", nil),
						Extensions:  newExt("", "route_types=1,2", "route_types=1,2"),
					}},
					&pref{Value: &param{
						Name:        "operator_onestop_id",
						In:          "query",
						Description: `Search for records by operator OnestopID`,
						Schema:      newSRVal("string", "", nil),
						Extensions:  newExt("", "operator_onestop_id=...", "operator_onestop_id=o-9q9-caltrain"),
					}},
					newPRef("includeAlertsParam"),
					&pref{Value: &param{
						Name:        "include_geometry",
						In:          "query",
						Description: `Include route geometry`,
						Schema:      newSRVal("string", "", []any{"true", "false"}),
						Extensions:  newExt("", "include_geometry=true", ""),
					}},
					&pref{Value: &param{
						Name:        "include_stops",
						In:          "query",
						Description: `Include route stops`,
						Schema:      newSRVal("string", "", []any{"true", "false"}),
						Extensions:  newExt("", "include_stops=true", ""),
					}},
					newPRef("idParam"),
					newPRef("afterParam"),
					newPRefExt("limitParam", "", "limit=1", ""),
					newPRefExt("formatParam", "", "format=png", "?format=png&feed_onestop_id=f-dr5r7-nycdotsiferry"),
					newPRefExt("searchParam", "", "search=daly+city", "?search=daly+city"),
					newPRefExt("onestopParam", "", "onestop_id=r-9q9j-l1", "onestop_id=r-9q9j-l1"),
					newPRefExt("sha1Param", "", "feed_version_sha1=041ffeec...", "feed_version_sha1=041ffeec98316e560bc2b91960f7150ad329bd5f"),
					newPRefExt("feedParam", "", "feed_onestop_id=f-sf~bay~area~rg", ""),
					newPRefExt("radiusParam", "Search for routes geographically, based on stops at this location; radius is in meters, requires lon and lat", "lon=-122&lat=37&radius=1000", "lon=-122.3&lat=37.8&radius=1000"),
					newPRef("latParam"),
					newPRef("lonParam"),
					newPRefExt("bboxParam", "", "bbox=-122.269,37.807,-122.267,37.808", ""),
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

// ResponseKey returns the GraphQL response entity key.
func (r RouteRequest) ResponseKey() string { return "routes" }

// Query returns a GraphQL query string and variables.
func (r RouteRequest) Query(ctx context.Context) (string, map[string]interface{}) {
	// These formats will need geometries included
	if r.ID > 0 || r.Format == "geojson" || r.Format == "geojsonl" || r.Format == "png" {
		r.IncludeGeometry = true
	}

	// Handle operator key
	if r.AgencyKey == "" {
		// pass
	} else if v, err := strconv.Atoi(r.AgencyKey); err == nil {
		r.AgencyID = v
	} else {
		r.OperatorOnestopID = r.AgencyKey
	}
	// Handle route key
	if r.RouteKey == "" {
		// pass
	} else if fsid, eid, ok := strings.Cut(r.RouteKey, ":"); ok {
		r.FeedOnestopID = fsid
		r.RouteID = eid
		r.IncludeGeometry = true
		r.IncludeStops = true
	} else if v, err := strconv.Atoi(r.RouteKey); err == nil {
		r.ID = v
		r.IncludeGeometry = true
		r.IncludeStops = true
	} else {
		r.OnestopID = r.RouteKey
		r.IncludeGeometry = true
		r.IncludeStops = true
	}

	where := hw{}
	if r.FeedVersionSHA1 != "" {
		where["feed_version_sha1"] = r.FeedVersionSHA1
	}
	if r.FeedOnestopID != "" {
		where["feed_onestop_id"] = r.FeedOnestopID
	}
	if r.RouteID != "" {
		where["route_id"] = r.RouteID
	}
	if r.RouteType != "" {
		where["route_type"] = r.RouteType
	}
	if r.RouteTypes != "" {
		where["route_types"] = commaSplit(r.RouteTypes)
	}
	if r.OnestopID != "" {
		where["onestop_id"] = r.OnestopID
	}
	if r.OperatorOnestopID != "" {
		where["operator_onestop_id"] = r.OperatorOnestopID
	}
	if r.AgencyID > 0 {
		where["agency_ids"] = []int{r.AgencyID}
	}
	if r.Lat != 0.0 && r.Lon != 0.0 {
		where["near"] = hw{"lat": r.Lat, "lon": r.Lon, "radius": r.Radius}
	}
	if r.Bbox != nil {
		where["bbox"] = r.Bbox.AsJson()
	}
	if r.Search != "" {
		where["search"] = r.Search
	}
	where["license"] = checkLicenseFilter(r.LicenseFilter)
	return routeQuery, hw{
		"limit":            r.CheckLimit(),
		"after":            r.CheckAfter(),
		"ids":              checkIds(r.ID),
		"where":            where,
		"include_alerts":   r.IncludeAlerts,
		"include_geometry": r.IncludeGeometry,
		"include_stops":    r.IncludeStops,
	}
}

//////////

type RouteKeyRequest struct {
	RouteRequest
}

func (r RouteKeyRequest) RequestInfo() RequestInfo {
	return RequestInfo{
		Path: "/routes/{route_key}",
		Get: RequestOperation{
			Query: routeQuery,
			Operation: &oa.Operation{
				Summary: `Search for routes`,
				Parameters: oa.Parameters{
					&pref{Value: &param{
						Name:        "route_key",
						In:          "path",
						Description: `Route lookup key; can be an integer ID, a '<feed onestop_id>:<gtfs route_id>' key, or a Onestop ID`,
						Schema:      newSRVal("string", "", nil),
					}},
					newPRef("limitParam"),
					newPRef("formatParam"),
					newPRef("includeAlertsParam"),
				},
			},
		},
	}
}

//////////

// type AgencyRouteRequest struct {
// 	RouteRequest
// }

// func (r AgencyRouteRequest) RequestInfo() RequestInfo {
// 	// Include all base parameters except for agency_key
// 	baseInfo := RouteRequest{}.RequestInfo()
// 	var params oa.Parameters
// 	params = append(params, &pref{Value: &param{
// 		Name:        "agency_key",
// 		In:          "path",
// 		Description: `Agency lookup key; can be an integer ID, a '<feed onestop_id>:<gtfs agency_id>' key, or a Onestop ID`,
// 		Schema:      newSRVal("string", "", nil),
// 	}})
// 	for _, param := range baseInfo.PathItem.Get.Parameters {
// 		if param.Value != nil && param.Value.Name == "agency_key" {
// 			continue
// 		}
// 		params = append(params, param)
// 	}
// 	return RequestInfo{
// 		Path: "/agencies/{agency_key}/routes",
// 		Get: RequestOperation{
// 			Query: routeQuery,
// 			Operation: &oa.Operation{
// 				Summary:     "Routes",
// 				Description: `Search for routes`,
// 				Parameters:  params,
// 			},
// 		},
// 	}
// }
