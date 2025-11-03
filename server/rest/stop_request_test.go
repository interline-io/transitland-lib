package rest

import (
	"context"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestStopRequest(t *testing.T) {
	fv := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
	osid := "s-9q8yyufxmv-sanfranciscocaltrain"
	bartstops := []string{"12TH", "16TH", "19TH", "19TH_N", "24TH", "ANTC", "ASHB", "BALB", "BAYF", "CAST", "CIVC", "COLS", "COLM", "CONC", "DALY", "DBRK", "DUBL", "DELN", "PLZA", "EMBR", "FRMT", "FTVL", "GLEN", "HAYW", "LAFY", "LAKE", "MCAR", "MCAR_S", "MLBR", "MONT", "NBRK", "NCON", "OAKL", "ORIN", "PITT", "PCTR", "PHIL", "POWL", "RICH", "ROCK", "SBRN", "SFIA", "SANL", "SHAY", "SSAN", "UCTY", "WCRK", "WARM", "WDUB", "WOAK"}
	caltrainRailStops := []string{"70011", "70012", "70021", "70022", "70031", "70032", "70041", "70042", "70051", "70052", "70061", "70062", "70071", "70072", "70081", "70082", "70091", "70092", "70101", "70102", "70111", "70112", "70121", "70122", "70131", "70132", "70141", "70142", "70151", "70152", "70161", "70162", "70171", "70172", "70191", "70192", "70201", "70202", "70211", "70212", "70221", "70222", "70231", "70232", "70241", "70242", "70251", "70252", "70261", "70262", "70271", "70272", "70281", "70282", "70291", "70292", "70301", "70302", "70311", "70312", "70321", "70322"}
	caltrainBusStops := []string{"777402", "777403"}
	_ = caltrainRailStops
	_ = caltrainBusStops
	testcases := []testCase{
		{
			name:         "basic",
			h:            StopRequest{},
			selector:     "stops.#.stop_id",
			expectSelect: nil,
			expectLength: 20,
		},
		// default
		{
			name:         "onestop_id",
			h:            StopRequest{OnestopID: osid},
			selector:     "stops.#.onestop_id",
			expectSelect: []string{osid},
		}, // default
		{
			name:         "stop_id",
			h:            StopRequest{StopID: "70011"},
			selector:     "stops.#.stop_id",
			expectSelect: []string{"70011"},
		}, // default
		{
			name:         "limit:1",
			h:            StopRequest{WithCursor: WithCursor{Limit: 1}},
			selector:     "stops.#.stop_id",
			expectSelect: nil,
			expectLength: 1,
		},
		{
			name:         "limit:100",
			h:            StopRequest{WithCursor: WithCursor{Limit: 100}},
			selector:     "stops.#.stop_id",
			expectSelect: nil,
			expectLength: 100,
		},
		{
			name:         "limit:1000",
			h:            StopRequest{WithCursor: WithCursor{Limit: 1000}},
			selector:     "stops.#.stop_id",
			expectSelect: nil,
			expectLength: 1000,
		},
		{
			name:         "feed_onestop_id",
			h:            StopRequest{FeedOnestopID: "BA", WithCursor: WithCursor{Limit: 100}},
			selector:     "stops.#.stop_id",
			expectSelect: bartstops,
		},
		{
			name:         "feed_onestop_id,stop_id",
			h:            StopRequest{FeedOnestopID: "BA", StopID: "12TH"},
			selector:     "stops.#.stop_id",
			expectSelect: []string{"12TH"},
		},
		{
			name:         "feed_version_sha1",
			h:            StopRequest{FeedVersionSHA1: fv},
			selector:     "stops.#.stop_id",
			expectSelect: nil,
			expectLength: 20,
		},
		{
			name:         "feed_version_sha1,limit:100",
			h:            StopRequest{FeedVersionSHA1: fv, WithCursor: WithCursor{Limit: 100}},
			selector:     "stops.#.stop_id",
			expectSelect: nil,
			expectLength: 50,
		},
		// {"served_by_route_types=1", StopRequest{ServedByRouteTypes: []int{1}, WithCursor: WithCursor{Limit:100}}, "", "stops.#.stop_id", bartstops, 0},
		// {"served_by_route_types=2", StopRequest{ServedByRouteTypes: []int{2}, WithCursor: WithCursor{Limit:100}}, "", "stops.#.stop_id", caltrainRailStops, 0},
		// {"served_by_route_types=3", StopRequest{ServedByRouteTypes: []int{3}, WithCursor: WithCursor{Limit:100}}, "", "stops.#.stop_id", caltrainBusStops, 0},
		{
			name:         "served_by_onestop_ids=o-9q9-bayarearapidtransit",
			h:            StopRequest{ServedByOnestopIds: "o-9q9-bayarearapidtransit", WithCursor: WithCursor{Limit: 100}},
			selector:     "stops.#.stop_id",
			expectSelect: bartstops,
		},
		{
			name:         "served_by_onestop_ids=o-9q9-bayarearapidtransit,o-9q9-caltrain",
			h:            StopRequest{ServedByOnestopIds: "o-9q9-bayarearapidtransit,o-9q9-caltrain", WithCursor: WithCursor{Limit: 1000}},
			selector:     "stops.#.stop_id",
			expectLength: 114,
		},
		{
			name:         "served_by_route_types=0,4",
			h:            StopRequest{ServedByRouteTypes: "0,4", FeedOnestopID: "HA", WithCursor: WithCursor{Limit: 1000}},
			selector:     "stops.#.stop_id",
			expectLength: 23,
		},
		{
			name:         "served_by_route_types=0",
			h:            StopRequest{ServedByRouteTypes: "0", FeedOnestopID: "HA", WithCursor: WithCursor{Limit: 1000}},
			selector:     "stops.#.stop_id",
			expectLength: 11,
		},
		{
			name:         "served_by_route_type=0",
			h:            StopRequest{ServedByRouteType: ptr(0), FeedOnestopID: "HA", WithCursor: WithCursor{Limit: 1000}},
			selector:     "stops.#.stop_id",
			expectLength: 11,
		},
		{
			name:         "served_by_route_type=1",
			h:            StopRequest{ServedByRouteType: ptr(1), FeedOnestopID: "BA", WithCursor: WithCursor{Limit: 1000}},
			selector:     "stops.#.stop_id",
			expectLength: 50,
		},
		{
			name:         "served_by_route_type=2",
			h:            StopRequest{ServedByRouteType: ptr(2), FeedOnestopID: "CT", WithCursor: WithCursor{Limit: 1000}},
			selector:     "stops.#.stop_id",
			expectLength: 62,
		},
		{
			name:         "served_by_route_type=3",
			h:            StopRequest{ServedByRouteType: ptr(3), FeedOnestopID: "CT", WithCursor: WithCursor{Limit: 1000}},
			selector:     "stops.#.stop_id",
			expectLength: 2,
		},
		// {"served_by_onestop_ids=o-9q9-caltrain,served_by_route_types=3", StopRequest{ServedByOnestopIds: []string{"o-9q9-caltrain"}, ServedByRouteTypes: []int{3}, WithCursor: WithCursor{Limit:100}}, "", "stops.#.stop_id", caltrainBusStops, 0},
		{
			name:         "lat,lon,radius 10m",
			h:            StopRequest{Lon: -122.407974, Lat: 37.784471, Radius: 10},
			selector:     "stops.#.stop_id",
			expectSelect: []string{"POWL"},
		},
		{
			name:         "lat,lon,radius 2000m",
			h:            StopRequest{Lon: -122.407974, Lat: 37.784471, Radius: 2000},
			selector:     "stops.#.stop_id",
			expectSelect: []string{"70011", "70012", "CIVC", "EMBR", "MONT", "POWL"},
		},
		{
			name:         "bbox",
			h:            StopRequest{Bbox: &restBbox{model.BoundingBox{MinLon: -122.2698781543005, MinLat: 37.80700393130445, MaxLon: -122.2677640139239, MaxLat: 37.8088734037938}}},
			selector:     "stops.#.stop_id",
			expectSelect: []string{"19TH", "19TH_N"},
		},
		{
			name:         "search",
			h:            StopRequest{Search: "macarthur"},
			selector:     "stops.#.stop_id",
			expectSelect: []string{"MCAR", "MCAR_S"},
		}, // default
		{
			name:         "feed:stop_id",
			h:            StopRequest{StopKey: "BA:FTVL"},
			selector:     "stops.#.stop_id",
			expectSelect: []string{"FTVL"},
		},
		{
			name:         "place",
			h:            StopRequest{StopKey: "BA:FTVL"},
			selector:     "stops.#.place.adm1_name",
			expectSelect: []string{"California"},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}

func TestStopRequest_IncludeRoutes(t *testing.T) {
	cfg := testconfig.Config(t, testconfig.Options{})
	testcases := []testCase{
		{
			name:         "no auth",
			h:            StopRequest{StopID: "70011", IncludeRoutes: true},
			selector:     "stops.0.route_stops",
			expectLength: 0,
		},
		{
			name:         "with extended user role",
			h:            StopRequest{StopID: "70011", IncludeRoutes: true},
			selector:     "stops.0.route_stops",
			expectLength: 5,
			user:         "test",
			userRoles:    []string{cfg.Roles.ExtendedUserRole},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}

func TestStopRequest_AdminCache(t *testing.T) {
	tc := testCase{
		name:         "place",
		h:            StopRequest{StopKey: "BA:FTVL"},
		selector:     "stops.#.place.adm1_name",
		expectSelect: []string{"California"},
	}
	type canLoadAdmins interface {
		LoadAdmins(context.Context) error
	}
	cfg := testconfig.Config(t, testconfig.Options{})
	if v, ok := cfg.Finder.(canLoadAdmins); !ok {
		t.Fatal("finder cant load admins")
	} else {
		if err := v.LoadAdmins(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	checkTestCase(t, tc)
}

func TestStopRequest_Format(t *testing.T) {
	tcs := []testCase{
		{
			name:   "stop geojson",
			format: "geojson",
			h:      StopRequest{FeedOnestopID: "CT", Format: "geojson", WithCursor: WithCursor{Limit: 20}},
			f: func(t *testing.T, jj string) {
				a := gjson.Get(jj, "features").Array()
				assert.Equal(t, 20, len(a))
				assert.Equal(t, "Feature", gjson.Get(jj, "features.0.type").String())
				assert.Equal(t, "Point", gjson.Get(jj, "features.0.geometry.type").String())
				assert.Equal(t, "CT", gjson.Get(jj, "features.0.properties.feed_version.feed.onestop_id").String())
				assert.Greater(t, gjson.Get(jj, "meta.after").Int(), int64(0))
			},
		},
		{
			name:   "stop geojsonl",
			format: "geojsonl",
			h:      StopRequest{FeedOnestopID: "CT", Format: "geojsonl", WithCursor: WithCursor{Limit: 20}},
			f: func(t *testing.T, jj string) {
				split := strings.Split(jj, "\n")
				assert.Equal(t, 20, len(split))
				assert.Equal(t, "Feature", gjson.Get(split[0], "type").String())
				assert.Equal(t, "Point", gjson.Get(split[0], "geometry.type").String())
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

func TestStopRequest_Pagination(t *testing.T) {
	cfg := testconfig.Config(t, testconfig.Options{})
	allEnts, err := cfg.Finder.FindStops(model.WithConfig(context.Background(), cfg), nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	allIds := []string{}
	for _, ent := range allEnts {
		allIds = append(allIds, ent.StopID.Val)
	}
	testcases := []testCase{
		{
			name:         "pagination exists",
			h:            StopRequest{},
			selector:     "meta.after",
			expectSelect: nil,
			expectLength: 1,
		},
		// just check presence
		{
			name:         "pagination limit 10",
			h:            StopRequest{WithCursor: WithCursor{Limit: 10}},
			selector:     "stops.#.stop_id",
			expectSelect: allIds[:10],
		},
		{
			name:         "pagination after 10",
			h:            StopRequest{WithCursor: WithCursor{Limit: 10, After: allEnts[10].ID}},
			selector:     "stops.#.stop_id",
			expectSelect: allIds[11:21],
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}

func TestStopRequest_License(t *testing.T) {
	testcases := []testCase{
		{
			name:         "license:share_alike_optional yes",
			h:            StopRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseShareAlikeOptional: "yes"}},
			selector:     "stops.#.stop_id",
			expectSelect: nil,
			expectLength: 2349,
		},
		{
			name:         "license:share_alike_optional no",
			h:            StopRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseShareAlikeOptional: "no"}},
			selector:     "stops.#.stop_id",
			expectSelect: nil,
			expectLength: 50,
		},
		{
			name:         "license:share_alike_optional exclude_no",
			h:            StopRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseShareAlikeOptional: "exclude_no"}},
			selector:     "stops.#.stop_id",
			expectSelect: nil,
			expectLength: 2413,
		},
		{
			name:         "license:commercial_use_allowed yes",
			h:            StopRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseCommercialUseAllowed: "yes"}},
			selector:     "stops.#.stop_id",
			expectSelect: nil,
			expectLength: 2349,
		},
		{
			name:         "license:commercial_use_allowed no",
			h:            StopRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseCommercialUseAllowed: "no"}},
			selector:     "stops.#.stop_id",
			expectSelect: nil,
			expectLength: 50,
		},
		{
			name:         "license:commercial_use_allowed exclude_no",
			h:            StopRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseCommercialUseAllowed: "exclude_no"}},
			selector:     "stops.#.stop_id",
			expectSelect: nil,
			expectLength: 2413,
		},
		{
			name:         "license:create_derived_product yes",
			h:            StopRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseCreateDerivedProduct: "yes"}},
			selector:     "stops.#.stop_id",
			expectSelect: nil,
			expectLength: 2349,
		},
		{
			name:         "license:create_derived_product no",
			h:            StopRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseCreateDerivedProduct: "no"}},
			selector:     "stops.#.stop_id",
			expectSelect: nil,
			expectLength: 50,
		},
		{
			name:         "license:create_derived_product exclude_no",
			h:            StopRequest{WithCursor: WithCursor{Limit: 10_000}, LicenseFilter: LicenseFilter{LicenseCreateDerivedProduct: "exclude_no"}},
			selector:     "stops.#.stop_id",
			expectSelect: nil,
			expectLength: 2413,
		},
		{
			name: "include_alerts:true",
			h:    StopRequest{StopKey: "BA:FTVL", IncludeAlerts: true},
			f: func(t *testing.T, jj string) {
				a := gjson.Get(jj, "stops.0.alerts").Array()
				assert.Equal(t, 2, len(a), "alert count")
			},
		},
		{
			name: "include_alerts:false",
			h:    StopRequest{StopKey: "BA:FTVL", IncludeAlerts: false},
			f: func(t *testing.T, jj string) {
				a := gjson.Get(jj, "stops.0.alerts").Array()
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
