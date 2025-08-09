package rest

import (
	"context"
	_ "embed"
	"strconv"
	"strings"

	oa "github.com/getkin/kin-openapi/openapi3"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-mw/auth/authn"
)

//go:embed stop_request.gql
var stopQuery string

// StopRequest holds options for a /stops request
type StopRequest struct {
	ID                 int       `json:"id,string"`
	StopKey            string    `json:"stop_key"`
	StopID             string    `json:"stop_id"`
	OnestopID          string    `json:"onestop_id"`
	FeedVersionSHA1    string    `json:"feed_version_sha1"`
	FeedOnestopID      string    `json:"feed_onestop_id"`
	Search             string    `json:"search"`
	Bbox               *restBbox `json:"bbox"`
	Lon                float64   `json:"lon,string"`
	Lat                float64   `json:"lat,string"`
	Radius             float64   `json:"radius,string"`
	Format             string    `json:"format"`
	ServedByOnestopIds string    `json:"served_by_onestop_ids"`
	ServedByRouteType  *int      `json:"served_by_route_type,string"`
	ServedByRouteTypes string    `json:"served_by_route_types"`
	IncludeAlerts      bool      `json:"include_alerts,string"`
	IncludeRoutes      bool      `json:"include_routes,string"`
	LicenseFilter
	WithCursor
}

func (r StopRequest) RequestInfo() RequestInfo {
	return RequestInfo{
		Path: "/stops",
		Get: RequestOperation{
			Query: stopQuery,
			Operation: &oa.Operation{
				Summary: `Search for stops`,
				Extensions: map[string]any{
					"x-alternates": []RequestAltPath{
						{"GET", "/stops.{format}", "Request stops in specified format"},
						{"GET", "/stops/{route_key}", "Request a stop by ID or Onestop ID"},
						{"GET", "/stops/{route_key}.format", "Request a stop by ID or Onestop ID in specified format"},
					},
				},
				Parameters: oa.Parameters{
					&pref{Value: &param{
						Name:        "stop_key",
						In:          "query",
						Description: `Stop lookup key; can be an integer ID, a '<feed onestop_id>:<gtfs stop_id>' key, or a Onestop ID`,
						Schema:      newSRVal("string", "", nil),
					}},
					&pref{Value: &param{
						Name:        "stop_id",
						In:          "query",
						Description: `Search for records with this GTFS stop_id`,
						Schema:      newSRVal("string", "", nil),
						Extensions:  newExt("", "stop_id=EMBR", "feed_onestop_id=f-c20-trimet&stop_id=1108"),
					}},
					&pref{Value: &param{
						Name:        "served_by_onestop_ids",
						In:          "query",
						Description: `Search stops visited by a route or agency OnestopID. Accepts comma separated values.`,
						Schema:      newSRVal("string", "", nil),
						Extensions:  newExt("", "served_by_onestop_ids=o-9q9-bart,o-9q9-caltrain", "served_by_onestop_ids=o-9q9-bart,o-9q9-caltrain"),
					}},
					&pref{Value: &param{
						Name:        "served_by_route_type",
						In:          "query",
						Description: `Search for stops served by a particular route (vehicle) type.`,
						Schema:      newSRVal("integer", "", nil),
						Extensions:  newExt("", "served_by_route_type=1", "served_by_route_type=1"),
					}},
					&pref{Value: &param{
						Name:        "served_by_route_types",
						In:          "query",
						Description: `Search for stops served by particular route (vehicle) types. Accepts comma separated values.`,
						Schema:      newSRVal("string", "", nil),
						Extensions:  newExt("", "served_by_route_types=1,2", "served_by_route_types=1,2"),
					}},
					newPRef("includeAlertsParam"),
					newPRef("idParam"),
					newPRef("afterParam"),
					newPRefExt("limitParam", "", "limit=1", ""),
					newPRefExt("formatParam", "", "format=geojson", ""),
					newPRefExt("searchParam", "", "search=embarcadero", ""),
					newPRefExt("onestopParam", "", "onestop_id=...", "onestop_id=s-9q8yyzcny3-embarcadero"),
					newPRefExt("sha1Param", "", "feed_version_sha1=1c4721d4...", "feed_version_sha1=1c4721d4e0c9fae1e81f7c79660696e4280ed05b"),
					newPRefExt("feedParam", "", "feed_onestop_id=f-c20-trimet", ""),
					newPRefExt("radiusParam", "Search for stops geographically; radius is in meters, requires lon and lat", "lon=-122.3&lat=37.8&radius=1000", ""),
					newPRef("lonParam"),
					newPRef("latParam"),
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
func (r StopRequest) ResponseKey() string { return "stops" }

// Query returns a GraphQL query string and variables.
func (r StopRequest) Query(ctx context.Context) (string, map[string]any) {
	if r.StopKey == "" {
		// pass
	} else if fsid, eid, ok := strings.Cut(r.StopKey, ":"); ok {
		r.FeedOnestopID = fsid
		r.StopID = eid
		r.IncludeRoutes = true
	} else if v, err := strconv.Atoi(r.StopKey); err == nil {
		r.ID = v
		r.IncludeRoutes = true
	} else {
		r.OnestopID = r.StopKey
		r.IncludeRoutes = true
	}

	user := authn.ForContext(ctx)
	if user == nil || (!user.HasRole("tl_user_pro") && r.IncludeRoutes) {
		log.For(ctx).Trace().Msg("setting include_routes = false")
		r.IncludeRoutes = false
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
	if r.Bbox != nil {
		where["bbox"] = r.Bbox.AsJson()
	}
	if r.Search != "" {
		where["search"] = r.Search
	}
	if r.ServedByOnestopIds != "" {
		where["served_by_onestop_ids"] = commaSplit(r.ServedByOnestopIds)
	}
	if r.ServedByRouteType != nil {
		where["served_by_route_type"] = *r.ServedByRouteType
	}
	if r.ServedByRouteTypes != "" {
		where["served_by_route_types"] = commaSplit(r.ServedByRouteTypes)
	}
	where["license"] = checkLicenseFilter(r.LicenseFilter)
	return stopQuery, hw{
		"limit":          r.CheckLimit(),
		"after":          r.CheckAfter(),
		"ids":            checkIds(r.ID),
		"include_alerts": r.IncludeAlerts,
		"include_routes": r.IncludeRoutes,
		"where":          where,
	}
}

///////////////

type StopEntityRequest struct {
	StopRequest
}

func (r StopEntityRequest) RequestInfo() RequestInfo {
	return RequestInfo{
		Path: "/stops/{stop_key}",
		Get: RequestOperation{
			Query: stopQuery,
			Operation: &oa.Operation{
				Summary: `Search for stops`,
				Parameters: oa.Parameters{
					&pref{Value: &param{
						Name:        "stop_key",
						In:          "query",
						Description: `Stop lookup key; can be an integer ID, a '<feed onestop_id>:<gtfs stop_id>' key, or a Onestop ID`,
						Schema:      newSRVal("string", "", nil),
					}},
					newPRef("includeAlertsParam"),
					newPRef("limitParam"),
					newPRef("formatParam"),
				},
			},
		},
	}
}
