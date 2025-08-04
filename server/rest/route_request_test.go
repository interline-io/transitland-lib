package rest

import (
	"context"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/model"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func ptr[T any](v T) *T {
	return &v
}

func TestRouteRequest(t *testing.T) {
	routeIds := []string{"1", "12", "14", "15", "16", "17", "19", "20", "24", "25", "275", "30", "31", "32", "33", "34", "35", "36", "360", "37", "38", "39", "400", "42", "45", "46", "48", "5", "51", "6", "60", "7", "75", "8", "9", "96", "97", "570", "571", "572", "573", "574", "800", "PWT", "SKY", "01", "03", "05", "07", "11", "19", "Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"}
	fv := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
	testcases := []testCase{
		{
			name:         "none",
			h:            RouteRequest{WithCursor: WithCursor{Limit: 1000}},
			selector:     "routes.#.route_id",
			expectSelect: routeIds,
		},
		{
			name:         "search",
			h:            RouteRequest{Search: "bullet"},
			selector:     "routes.#.route_id",
			expectSelect: []string{"Bu-130"},
		},
		{
			name:         "feed_onestop_id",
			h:            RouteRequest{FeedOnestopID: "CT"},
			selector:     "routes.#.route_id",
			expectSelect: []string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
		},
		{

			name:         "route_type:2",
			h:            RouteRequest{RouteType: "2"},
			selector:     "routes.#.route_id",
			expectSelect: []string{"Bu-130", "Li-130", "Lo-130", "Gi-130", "Sp-130"},
		},
		{
			name:         "route_type:1",
			h:            RouteRequest{RouteType: "1"},
			selector:     "routes.#.route_id",
			expectSelect: []string{"01", "03", "05", "07", "11", "19"},
		},
		{
			name:         "route_types:1,4",
			h:            RouteRequest{RouteTypes: "1,4"},
			selector:     "routes.#.route_id",
			expectSelect: []string{"01", "03", "05", "07", "11", "19", "PWT"},
		},
		{
			name:         "feed_onestop_id,route_id",
			h:            RouteRequest{FeedOnestopID: "BA", RouteID: "19"},
			selector:     "routes.#.route_id",
			expectSelect: []string{"19"},
		},
		{
			name:         "feed_version_sha1",
			h:            RouteRequest{FeedVersionSHA1: fv},
			selector:     "routes.#.feed_version.sha1",
			expectSelect: []string{fv, fv, fv, fv, fv, fv},
		},
		{
			name:         "operator_onestop_id",
			h:            RouteRequest{OperatorOnestopID: "o-9q9-bayarearapidtransit"},
			selector:     "routes.#.route_id",
			expectSelect: []string{"01", "03", "05", "07", "11", "19"},
		},
		{
			name:         "lat,lon,radius 100m",
			h:            RouteRequest{Lon: -122.407974, Lat: 37.784471, Radius: 100},
			selector:     "routes.#.route_id",
			expectSelect: []string{"01", "05", "07", "11"},
		},
		{
			name:         "lat,lon,radius 2000m",
			h:            RouteRequest{Lon: -122.407974, Lat: 37.784471, Radius: 2000},
			selector:     "routes.#.route_id",
			expectSelect: []string{"Bu-130", "Li-130", "Lo-130", "Gi-130", "Sp-130", "01", "05", "07", "11"},
		},
		{
			name:         "bbox",
			h:            RouteRequest{Bbox: &restBbox{model.BoundingBox{MinLon: -122.2698781543005, MinLat: 37.80700393130445, MaxLon: -122.2677640139239, MaxLat: 37.8088734037938}}},
			selector:     "routes.#.route_id",
			expectSelect: []string{"01", "03", "07"},
		},
		{
			name:         "feed:route_id",
			h:            RouteRequest{RouteKey: "BA:01"},
			selector:     "routes.#.route_id",
			expectSelect: []string{"01"},
		},
		{
			name: "include_alerts:true",
			h:    RouteRequest{RouteKey: "BA:05", IncludeAlerts: true},
			f: func(t *testing.T, jj string) {
				a := gjson.Get(jj, "routes.0.alerts").Array()
				assert.Equal(t, 2, len(a), "alert count")
			},
		},
		{
			name: "include_alerts:false",
			h:    RouteRequest{RouteKey: "BA:05", IncludeAlerts: false},
			f: func(t *testing.T, jj string) {
				a := gjson.Get(jj, "routes.0.alerts").Array()
				assert.Equal(t, 0, len(a), "alert count")
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}

func TestRouteRequest_Format(t *testing.T) {
	tcs := []testCase{
		{
			name:   "route geojson",
			format: "geojson",
			h:      RouteRequest{FeedOnestopID: "CT", Format: "geojson", WithCursor: WithCursor{Limit: 5}},
			f: func(t *testing.T, jj string) {
				a := gjson.Get(jj, "features").Array()
				assert.Equal(t, 5, len(a))
				assert.Equal(t, "Feature", gjson.Get(jj, "features.0.type").String())
				assert.Equal(t, "MultiLineString", gjson.Get(jj, "features.0.geometry.type").String())
				assert.Equal(t, "CT", gjson.Get(jj, "features.0.properties.feed_version.feed.onestop_id").String())
				assert.Greater(t, gjson.Get(jj, "meta.after").Int(), int64(0))
			},
		},
		{
			name:   "route geojsonl",
			format: "geojsonl",
			h:      RouteRequest{FeedOnestopID: "CT", Format: "geojsonl", WithCursor: WithCursor{Limit: 5}},
			f: func(t *testing.T, jj string) {
				split := strings.Split(jj, "\n")
				assert.Equal(t, 5, len(split))
				assert.Equal(t, "Feature", gjson.Get(split[0], "type").String())
				assert.Equal(t, "MultiLineString", gjson.Get(split[0], "geometry.type").String())
				assert.Equal(t, "CT", gjson.Get(split[0], "properties.feed_version.feed.onestop_id").String())
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}

func TestRouteRequest_Pagination(t *testing.T) {
	cfg := testconfig.Config(t, testconfig.Options{})
	allEnts, err := cfg.Finder.FindRoutes(model.WithConfig(context.Background(), cfg), nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	allIds := []string{}
	for _, ent := range allEnts {
		allIds = append(allIds, ent.RouteID.Val)
	}
	testcases := []testCase{
		{
			name:         "limit:1",
			h:            RouteRequest{WithCursor: WithCursor{Limit: 1}},
			selector:     "routes.#.route_id",
			expectSelect: nil,
			expectLength: 1,
		},
		{
			name:         "limit:100",
			h:            RouteRequest{WithCursor: WithCursor{Limit: 100}},
			selector:     "routes.#.route_id",
			expectSelect: nil,
			expectLength: 57,
		},
		{
			name:         "pagination exists",
			h:            RouteRequest{},
			selector:     "meta.after",
			expectSelect: nil,
			expectLength: 1,
		}, // just check presence
		{
			name:         "pagination limit 10",
			h:            RouteRequest{WithCursor: WithCursor{Limit: 10}},
			selector:     "routes.#.route_id",
			expectSelect: allIds[:10],
		},
		{
			name:         "pagination after 10",
			h:            RouteRequest{WithCursor: WithCursor{Limit: 10, After: allEnts[10].ID}},
			selector:     "routes.#.route_id",
			expectSelect: allIds[11:21],
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}

func TestRouteRequest_License(t *testing.T) {
	testcases := []testCase{
		{
			name: "license:share_alike_optional yes",
			h:    RouteRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseShareAlikeOptional: "yes"}}, selector: "routes.#.route_id",
			expectLength: 45,
		},
		{
			name: "license:share_alike_optional no",
			h:    RouteRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseShareAlikeOptional: "no"}}, selector: "routes.#.route_id",
			expectLength: 6,
		},
		{
			name: "license:share_alike_optional exclude_no",
			h:    RouteRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseShareAlikeOptional: "exclude_no"}}, selector: "routes.#.route_id",
			expectLength: 51,
		},
		{
			name: "license:commercial_use_allowed yes",
			h:    RouteRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseCommercialUseAllowed: "yes"}}, selector: "routes.#.route_id",
			expectLength: 45,
		},
		{
			name: "license:commercial_use_allowed no",
			h:    RouteRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseCommercialUseAllowed: "no"}}, selector: "routes.#.route_id",
			expectLength: 6,
		},
		{
			name: "license:commercial_use_allowed exclude_no",
			h:    RouteRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseCommercialUseAllowed: "exclude_no"}}, selector: "routes.#.route_id",
			expectLength: 51,
		},
		{
			name: "license:create_derived_product yes",
			h:    RouteRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseCreateDerivedProduct: "yes"}}, selector: "routes.#.route_id",
			expectLength: 45,
		},
		{
			name: "license:create_derived_product no",
			h:    RouteRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseCreateDerivedProduct: "no"}}, selector: "routes.#.route_id",
			expectLength: 6,
		},
		{
			name: "license:create_derived_product exclude_no",
			h:    RouteRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseCreateDerivedProduct: "exclude_no"}}, selector: "routes.#.route_id",
			expectLength: 51,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}
