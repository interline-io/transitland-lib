package rest

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/model"
	"github.com/interline-io/transitland-lib/server/gql"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestTripRequest(t *testing.T) {
	cfg := testconfig.Config(t, testconfig.Options{
		WhenUtc: "2018-06-01T00:00:00Z",
		RTJsons: testconfig.DefaultRTJson(),
	})
	graphqlHandler, err := gql.NewServer()
	if err != nil {
		t.Fatal(err)
	}

	ctx := model.WithConfig(context.Background(), cfg)
	d, err := makeGraphQLRequest(ctx, graphqlHandler, `query{routes(where:{feed_onestop_id:"BA",route_id:"11"}) {id onestop_id}}`, nil)
	if err != nil {
		t.Error("failed to get route id for tests")
	}
	routeId := int(gjson.Get(toJson(d), "routes.0.id").Int())
	routeOnestopId := gjson.Get(toJson(d), "routes.0.onestop_id").String()
	d2, err := makeGraphQLRequest(ctx, graphqlHandler, `query{trips(where:{trip_id:"5132248WKDY"}){id}}`, nil)
	if err != nil {
		t.Error("failed to get route id for tests")
	}
	tripId := int(gjson.Get(toJson(d2), "trips.0.id").Int())
	fv := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
	ctfv := "d2813c293bcfd7a97dde599527ae6c62c98e66c6"
	testcases := []testCase{
		{
			name:         "none",
			h:            TripRequest{},
			selector:     "trips.#.trip_id",
			expectSelect: nil,
			expectLength: 20,
		},
		{
			name:         "feed_onestop_id",
			h:            TripRequest{FeedOnestopID: "BA"},
			selector:     "trips.#.trip_id",
			expectSelect: nil,
			expectLength: 20,
		},
		{
			name:         "feed_onestop_id ct",
			h:            TripRequest{FeedOnestopID: "CT", WithCursor: WithCursor{Limit: 1000}},
			selector:     "trips.#.trip_id",
			expectSelect: nil,
			expectLength: 185,
		},
		{
			name:         "feed_version_sha1",
			h:            TripRequest{FeedVersionSHA1: ctfv, WithCursor: WithCursor{Limit: 1000}},
			selector:     "trips.#.trip_id",
			expectSelect: nil,
			expectLength: 185,
		},
		{
			name:         "feed_version_sha1 ba",
			h:            TripRequest{FeedVersionSHA1: fv, WithCursor: WithCursor{Limit: 1000}},
			selector:     "trips.#.trip_id",
			expectSelect: nil,
			expectLength: 1000,
		}, // over 1000
		{
			name:         "trip_id",
			h:            TripRequest{TripID: "5132248WKDY"},
			selector:     "trips.#.trip_id",
			expectSelect: []string{"5132248WKDY"},
		},
		{
			name:         "trip_id,feed_version_id",
			h:            TripRequest{TripID: "5132248WKDY", FeedVersionSHA1: fv},
			selector:     "trips.#.trip_id",
			expectSelect: []string{"5132248WKDY"},
		},
		{
			name:         "route_id",
			h:            TripRequest{WithCursor: WithCursor{Limit: 1000}, RouteID: routeId},
			selector:     "trips.#.trip_id",
			expectSelect: nil,
			expectLength: 364,
		},
		{
			name:         "route_id,service_date 1",
			h:            TripRequest{WithCursor: WithCursor{Limit: 1000}, RouteID: routeId, ServiceDate: "2018-01-01"},
			selector:     "trips.#.trip_id",
			expectSelect: nil,
		},
		{
			name:         "route_id,service_date 2",
			h:            TripRequest{WithCursor: WithCursor{Limit: 1000}, RouteID: routeId, ServiceDate: "2019-01-01"},
			selector:     "trips.#.trip_id",
			expectSelect: nil,
			expectLength: 100,
		},
		{
			name:         "route_id,service_date 3",
			h:            TripRequest{WithCursor: WithCursor{Limit: 1000}, RouteID: routeId, ServiceDate: "2019-01-02"},
			selector:     "trips.#.trip_id",
			expectSelect: nil,
			expectLength: 152,
		},
		{
			name:         "route_id,service_date 4",
			h:            TripRequest{WithCursor: WithCursor{Limit: 1000}, RouteID: routeId, ServiceDate: "2020-05-18"},
			selector:     "trips.#.trip_id",
			expectSelect: nil,
		},
		{
			name:         "route_id,trip_id",
			h:            TripRequest{WithCursor: WithCursor{Limit: 1000}, RouteID: routeId, TripID: "5132248WKDY"},
			selector:     "trips.#.trip_id",
			expectSelect: []string{"5132248WKDY"},
		},
		{
			name:         "include_geometry=true",
			h:            TripRequest{TripID: "5132248WKDY", IncludeGeometry: true},
			selector:     "trips.0.shape.geometry.type",
			expectSelect: []string{"LineString"},
		},
		{
			name:         "include_geometry=false",
			h:            TripRequest{TripID: "5132248WKDY", IncludeGeometry: false},
			selector:     "trips.0.shape.geometry.type",
			expectSelect: []string{},
		},
		{
			name:         "does not include stop_times without id",
			h:            TripRequest{TripID: "5132248WKDY"},
			selector:     "trips.0.stop_times.#.stop_sequence",
			expectSelect: nil,
		},
		{
			name:         "id includes stop_times",
			h:            TripRequest{ID: tripId},
			selector:     "trips.0.stop_times.#.stop_sequence",
			expectSelect: nil,
			expectLength: 18,
		},
		{
			name:         "route_key onestop_id",
			h:            TripRequest{WithCursor: WithCursor{Limit: 1000}, RouteKey: routeOnestopId},
			selector:     "trips.#.trip_id",
			expectSelect: nil,
			expectLength: 364,
		},
		{
			name:         "route_key int",
			h:            TripRequest{WithCursor: WithCursor{Limit: 1000}, RouteKey: strconv.Itoa(routeId)},
			selector:     "trips.#.trip_id",
			expectSelect: nil,
			expectLength: 364,
		},
		{
			name: "include_alerts:true",
			h:    TripRequest{TripID: "1031527WKDY", IncludeAlerts: true},
			f: func(t *testing.T, jj string) {
				a := gjson.Get(jj, "trips.0.alerts").Array()
				assert.Equal(t, 2, len(a), "alert count")
			},
		},
		{
			name: "include_alerts:false",
			h:    TripRequest{TripID: "1031527WKDY", IncludeAlerts: false},
			f: func(t *testing.T, jj string) {
				a := gjson.Get(jj, "trips.0.alerts").Array()
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

func TestTripRequest_Format(t *testing.T) {
	tcs := []testCase{
		{
			name:   "trip geojson",
			format: "geojson",
			h:      TripRequest{TripID: "5132248WKDY", Format: "geojson", WithCursor: WithCursor{Limit: 1}},
			f: func(t *testing.T, jj string) {
				a := gjson.Get(jj, "features").Array()
				assert.Equal(t, 1, len(a))
				assert.Equal(t, "Feature", gjson.Get(jj, "features.0.type").String())
				assert.Equal(t, "LineString", gjson.Get(jj, "features.0.geometry.type").String())
				assert.Equal(t, "BA", gjson.Get(jj, "features.0.properties.feed_version.feed.onestop_id").String())
				assert.Greater(t, gjson.Get(jj, "meta.after").Int(), int64(0))
			},
		},
		{
			name:   "trip geojsonl",
			format: "geojsonl",
			h:      TripRequest{TripID: "5132248WKDY", Format: "geojsonl"},
			f: func(t *testing.T, jj string) {
				split := strings.Split(jj, "\n")
				assert.Equal(t, 1, len(split))
				assert.Equal(t, "Feature", gjson.Get(split[0], "type").String())
				assert.Equal(t, "LineString", gjson.Get(split[0], "geometry.type").String())
				assert.Equal(t, "BA", gjson.Get(split[0], "properties.feed_version.feed.onestop_id").String())
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}

func TestTripRequest_Pagination(t *testing.T) {
	testcases := []testCase{
		{
			name:         "limit:1",
			h:            TripRequest{WithCursor: WithCursor{Limit: 1}},
			selector:     "trips.#.trip_id",
			expectSelect: nil,
			expectLength: 1,
		},
		{
			name:         "limit:100",
			h:            TripRequest{WithCursor: WithCursor{Limit: 100}},
			selector:     "trips.#.trip_id",
			expectSelect: nil,
			expectLength: 100,
		},
		{
			name:         "limit:1000",
			h:            TripRequest{WithCursor: WithCursor{Limit: 1000}},
			selector:     "trips.#.trip_id",
			expectSelect: nil,
			expectLength: 1000,
		},
		{
			name:         "limit:10000",
			h:            TripRequest{WithCursor: WithCursor{Limit: 10_000}},
			selector:     "trips.#.trip_id",
			expectSelect: nil,
			expectLength: 10_000,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}

func TestTripRequest_License(t *testing.T) {
	testcases := []testCase{
		{
			name: "license:share_alike_optional yes",
			h:    TripRequest{WithCursor: WithCursor{Limit: 100_000}, LicenseFilter: LicenseFilter{LicenseShareAlikeOptional: "yes"}}, selector: "trips.#.trip_id",
			expectLength: 14718,
		},
		{
			name: "license:share_alike_optional no",
			h:    TripRequest{WithCursor: WithCursor{Limit: 100_000}, LicenseFilter: LicenseFilter{LicenseShareAlikeOptional: "no"}}, selector: "trips.#.trip_id",
			expectLength: 2525,
		},
		{
			name: "license:share_alike_optional exclude_no",
			h:    TripRequest{WithCursor: WithCursor{Limit: 100_000}, LicenseFilter: LicenseFilter{LicenseShareAlikeOptional: "exclude_no"}}, selector: "trips.#.trip_id",
			expectLength: 14903,
		},
		{
			name: "license:commercial_use_allowed yes",
			h:    TripRequest{WithCursor: WithCursor{Limit: 100_000}, LicenseFilter: LicenseFilter{LicenseCommercialUseAllowed: "yes"}}, selector: "trips.#.trip_id",
			expectLength: 14718,
		},
		{
			name: "license:commercial_use_allowed no",
			h:    TripRequest{WithCursor: WithCursor{Limit: 100_000}, LicenseFilter: LicenseFilter{LicenseCommercialUseAllowed: "no"}}, selector: "trips.#.trip_id",
			expectLength: 2525,
		},
		{
			name: "license:commercial_use_allowed exclude_no",
			h:    TripRequest{WithCursor: WithCursor{Limit: 100_000}, LicenseFilter: LicenseFilter{LicenseCommercialUseAllowed: "exclude_no"}}, selector: "trips.#.trip_id",
			expectLength: 14903,
		},
		{
			name: "license:create_derived_product yes",
			h:    TripRequest{WithCursor: WithCursor{Limit: 100_000}, LicenseFilter: LicenseFilter{LicenseCreateDerivedProduct: "yes"}}, selector: "trips.#.trip_id",
			expectLength: 14718,
		},
		{
			name: "license:create_derived_product no",
			h:    TripRequest{WithCursor: WithCursor{Limit: 100_000}, LicenseFilter: LicenseFilter{LicenseCreateDerivedProduct: "no"}}, selector: "trips.#.trip_id",
			expectLength: 2525,
		},
		{
			name: "license:create_derived_product exclude_no",
			h:    TripRequest{WithCursor: WithCursor{Limit: 100_000}, LicenseFilter: LicenseFilter{LicenseCreateDerivedProduct: "exclude_no"}}, selector: "trips.#.trip_id",
			expectLength: 14903,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}
