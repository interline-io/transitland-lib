package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/meters"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/proto"
)

// resolveID posts a GraphQL query to the handler as admin and returns the
// integer id found at the given gjson path. Used to exercise the
// download-by-integer-id paths, since fixture serial ids are not stable.
func resolveID(t *testing.T, gqlSrv http.Handler, query string, path string) string {
	req, _ := http.NewRequest("POST", "/", strings.NewReader(query))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	usercheck.AdminDefaultMiddleware("test")(gqlSrv).ServeHTTP(rr, req)
	id := gjson.Get(rr.Body.String(), path).String()
	if id == "" {
		t.Fatalf("could not resolve id at %q from response: %s", path, rr.Body.String())
	}
	return id
}

func TestFeedVersionDownloadRequest(t *testing.T) {
	gqlSrv, restSrv, _ := testHandlersWithOptions(t, testconfig.Options{
		Storage: testdata.Path("server", "tmp"),
	})

	t.Run("ok", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feed_versions/d2813c293bcfd7a97dde599527ae6c62c98e66c6/download", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 200 {
			t.Errorf("got status code %d, expected 200", sc)
		}
		if sc := len(rr.Body.Bytes()); sc != 59324 {
			t.Errorf("got %d bytes, expected 59324", sc)
		}
	})
	t.Run("ok by integer id", func(t *testing.T) {
		fvID := resolveID(t, gqlSrv,
			`{"query":"{feed_versions(where:{sha1:\"d2813c293bcfd7a97dde599527ae6c62c98e66c6\"}){id}}"}`,
			"data.feed_versions.0.id")
		req, _ := http.NewRequest("GET", "/feed_versions/"+fvID+"/download", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)
		assert.Equal(t, 200, rr.Result().StatusCode, "status code")
		assert.Equal(t, 59324, rr.Body.Len(), "body length")
	})
	t.Run("not authorized as anon", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feed_versions/d2813c293bcfd7a97dde599527ae6c62c98e66c6/download", nil)
		rr := httptest.NewRecorder()
		asAnon := restSrv
		asAnon.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 401 {
			t.Errorf("got status code %d, expected 401", sc)
		}
	})
	t.Run("not authorized as user, missing role", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feed_versions/d2813c293bcfd7a97dde599527ae6c62c98e66c6/download", nil)
		rr := httptest.NewRecorder()
		asUser := usercheck.NewUserDefaultMiddleware(func() authn.User {
			return authn.NewCtxUser("testuser", "", "").WithRoles("testrole")
		})(restSrv)
		asUser.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 401 {
			t.Errorf("got status code %d, expected 401", sc)
		}
	})
	t.Run("not authorized as user, only current download role", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feed_versions/d2813c293bcfd7a97dde599527ae6c62c98e66c6/download", nil)
		rr := httptest.NewRecorder()
		asUser := usercheck.NewUserDefaultMiddleware(func() authn.User {
			return authn.NewCtxUser("testuser", "", "").WithRoles("tl_download_fv_current")
		})(restSrv)
		asUser.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 401 {
			t.Errorf("got status code %d, expected 401", sc)
		}
	})
	t.Run("authorized as user", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feed_versions/d2813c293bcfd7a97dde599527ae6c62c98e66c6/download", nil)
		rr := httptest.NewRecorder()
		asUser := usercheck.NewUserDefaultMiddleware(func() authn.User {
			return authn.NewCtxUser("testuser", "", "").WithRoles("tl_download_fv_historic")
		})(restSrv)
		asUser.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 200 {
			t.Errorf("got status code %d, expected 200", sc)
		}
		if sc := len(rr.Body.Bytes()); sc != 59324 {
			t.Errorf("got %d bytes, expected 59324", sc)
		}
	})
	t.Run("not authorized as anon, not redistributable", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feed_versions/dd7aca4a8e4c90908fd3603c097fabee75fea907/download", nil)
		rr := httptest.NewRecorder()
		asAnon := restSrv
		asAnon.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 401 {
			t.Errorf("got status code %d, expected 401", sc)
		}
	})
	t.Run("not authorized as user, not redistributable", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feed_versions/dd7aca4a8e4c90908fd3603c097fabee75fea907/download", nil)
		rr := httptest.NewRecorder()
		asUser := usercheck.NewUserDefaultMiddleware(func() authn.User {
			return authn.NewCtxUser("testuser", "", "").WithRoles("tl_download_fv_historic")
		})(restSrv)
		asUser.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 401 {
			t.Errorf("got status code %d, expected 401", sc)
		}
	})
	t.Run("not found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feed_versions/asdxyz/download", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 404 {
			t.Errorf("got status code %d, expected 404", sc)
		}
	})
}

func TestFeedDownloadLatestRequest(t *testing.T) {
	gqlSrv, restSrv, _ := testHandlersWithOptions(t, testconfig.Options{
		Storage: testdata.Path("server", "tmp"),
	})

	t.Run("ok", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/CT/download_latest_feed_version", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 200 {
			t.Errorf("got status code %d, expected 200", sc)
		}
		if sc := len(rr.Body.Bytes()); sc != 59324 {
			t.Errorf("got %d bytes, expected 59324", sc)
		}
	})
	t.Run("ok by integer id", func(t *testing.T) {
		feedID := resolveID(t, gqlSrv,
			`{"query":"{feeds(where:{onestop_id:\"CT\"}){id}}"}`,
			"data.feeds.0.id")
		req, _ := http.NewRequest("GET", "/feeds/"+feedID+"/download_latest_feed_version", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)
		assert.Equal(t, 200, rr.Result().StatusCode, "status code")
		assert.Equal(t, 59324, rr.Body.Len(), "body length")
	})
	t.Run("ok as user", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/CT/download_latest_feed_version", nil)
		rr := httptest.NewRecorder()
		asUser := usercheck.NewUserDefaultMiddleware(func() authn.User {
			return authn.NewCtxUser("testuser", "", "").WithRoles("tl_download_fv_current")
		})(restSrv)
		asUser.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 200 {
			t.Errorf("got status code %d, expected 200", sc)
		}
		if sc := len(rr.Body.Bytes()); sc != 59324 {
			t.Errorf("got %d bytes, expected 59324", sc)
		}
	})
	t.Run("not authorized as anon", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/CT/download_latest_feed_version", nil)
		rr := httptest.NewRecorder()
		asAnon := restSrv
		asAnon.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 401 {
			t.Errorf("got status code %d, expected 401", sc)
		}
	})
	t.Run("not authorized as user, missing role", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/CT/download_latest_feed_version", nil)
		rr := httptest.NewRecorder()
		asUser := usercheck.NewUserDefaultMiddleware(func() authn.User {
			return authn.NewCtxUser("testuser", "", "").WithRoles("testrole")
		})(restSrv)
		asUser.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 401 {
			t.Errorf("got status code %d, expected 401", sc)
		}
	})
	t.Run("not authorized as user, not redistributable", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/BA/download_latest_feed_version", nil)
		rr := httptest.NewRecorder()
		asUser := usercheck.NewUserDefaultMiddleware(func() authn.User {
			return authn.NewCtxUser("testuser", "", "").WithRoles("download_latest_feed_version")
		})(restSrv)
		asUser.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 401 {
			t.Errorf("got status code %d, expected 401", sc)
		}
	})

	t.Run("not found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/asdxyz/download_latest_feed_version", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 404 {
			t.Errorf("got status code %d, expected 404", sc)
		}
	})
}

func TestFeedVersionDownloadQuota(t *testing.T) {
	_, restSrv, _ := testHandlersWithOptions(t, testconfig.Options{
		Storage: testdata.Path("server", "tmp"),
	})

	// Both download endpoints, and the is_latest_feed_version value each one
	// is expected to check and record.
	endpoints := []struct {
		name     string
		path     string
		isLatest string
	}{
		{"historical", "/feed_versions/d2813c293bcfd7a97dde599527ae6c62c98e66c6/download", "false"},
		{"latest", "/feeds/CT/download_latest_feed_version", "true"},
	}

	// The admin user clears both download roles, and leaving the "rest" meter
	// allowed means only the download quota can reject the request.
	get := func(mp *fakeMeterProvider, path string) *httptest.ResponseRecorder {
		req, _ := http.NewRequest("GET", path, nil)
		rr := httptest.NewRecorder()
		metered := meters.WithMeter(mp, "rest", 1.0, nil)(restSrv)
		usercheck.AdminDefaultMiddleware("test")(metered).ServeHTTP(rr, req)
		return rr
	}

	t.Run("within quota", func(t *testing.T) {
		for _, tc := range endpoints {
			t.Run(tc.name, func(t *testing.T) {
				mp := &fakeMeterProvider{}
				rr := get(mp, tc.path)
				assert.Equal(t, 200, rr.Result().StatusCode, "status code")
				assert.Equal(t, 59324, rr.Body.Len(), "body length")
				assert.Equal(t, 1, mp.eventCount(feedVersionDownloadMeter), "download events")
			})
		}
	})
	t.Run("over quota", func(t *testing.T) {
		for _, tc := range endpoints {
			t.Run(tc.name, func(t *testing.T) {
				mp := &fakeMeterProvider{allow: map[string]bool{feedVersionDownloadMeter: false}}
				rr := get(mp, tc.path)
				assert.Equal(t, 429, rr.Result().StatusCode, "status code")
				assert.Equal(t, "application/json", rr.Header().Get("Content-Type"), "content-type")
				// The zip must not be served alongside the error, and the quota
				// the download was rejected for must not be consumed.
				assert.JSONEq(t, `{"error":"too many requests"}`, rr.Body.String(), "body")
				assert.Equal(t, 0, mp.eventCount(feedVersionDownloadMeter), "download events")
			})
		}
	})
	t.Run("checked dimensions match the recorded event", func(t *testing.T) {
		// The quota is dimension-scoped, and a limit matches only when its own
		// dimensions are contained in the checked ones.
		for _, tc := range endpoints {
			t.Run(tc.name, func(t *testing.T) {
				mp := &fakeMeterProvider{}
				get(mp, tc.path)
				checked, ok := mp.lastCheck(feedVersionDownloadMeter)
				if !assert.True(t, ok, "download quota was checked") {
					return
				}
				assert.Equal(t, 1.0, checked.value, "checked value")
				assert.Equal(t, tc.isLatest, dimValue(checked.dims, "is_latest_feed_version"), "is_latest_feed_version")
				assert.Equal(t, "CT", dimValue(checked.dims, "feed_onestop_id"), "feed_onestop_id")
				assert.NotEmpty(t, dimValue(checked.dims, "fv_sha1"), "fv_sha1")
				assert.Equal(t, 1, mp.eventCount(feedVersionDownloadMeter), "download events")
				assert.Equal(t, checked.dims, mp.lastEvent(feedVersionDownloadMeter).Dimensions, "checked and metered dimensions")
			})
		}
	})
	t.Run("not found is not checked", func(t *testing.T) {
		// The quota is checked only once a download is actually going to be
		// served, so a miss cannot consume it.
		mp := &fakeMeterProvider{}
		rr := get(mp, "/feed_versions/asdxyz/download")
		assert.Equal(t, 404, rr.Result().StatusCode, "status code")
		_, ok := mp.lastCheck(feedVersionDownloadMeter)
		assert.False(t, ok, "download quota was not checked")
	})
}

func TestFeedDownloadRtLatestRequest(t *testing.T) {
	_, restSrv, _ := testHandlersWithOptions(t, testconfig.Options{
		Storage: testdata.Path("server", "tmp"),
		RTJsons: []testconfig.RTJsonFile{
			{Feed: "BA~rt", Ftype: "realtime_alerts", Fname: "BA-alerts.json"},
			{Feed: "BA~rt", Ftype: "realtime_trip_updates", Fname: "BA.json"},
		},
	})
	t.Run("ok as user", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/BA~rt/download_latest_rt/alerts.json", nil)
		rr := httptest.NewRecorder()
		asUser := usercheck.NewUserDefaultMiddleware(func() authn.User {
			return authn.NewCtxUser("testuser", "", "").WithRoles("tl_download_fv_current")
		})(restSrv)
		asUser.ServeHTTP(rr, req)
		assert.Equal(t, 200, rr.Result().StatusCode, "status code")
	})
	t.Run("ok as anon", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/BA~rt/download_latest_rt/alerts.json", nil)
		rr := httptest.NewRecorder()
		asAnon := restSrv
		asAnon.ServeHTTP(rr, req)
		assert.Equal(t, 200, rr.Result().StatusCode, "status code")
	})
	t.Run("alerts ok json", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/BA~rt/download_latest_rt/alerts.json", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)
		assert.Equal(t, "application/json", rr.Header().Get("content-type"), "content-type")
		assert.Equal(t, 200, rr.Result().StatusCode, "status code")
		var checkJson map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &checkJson); err != nil {
			t.Fatal(err)
		}
		if v, ok := checkJson["entity"].([]any); ok {
			assert.Greater(t, len(v), 0, "should have entities")
		} else {
			t.Fatal("expected entities")
		}
	})
	t.Run("alerts ok pb", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/BA~rt/download_latest_rt/alerts.pb", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)
		assert.Equal(t, "application/octet-stream", rr.Header().Get("content-type"), "content-type")
		assert.Equal(t, 200, rr.Result().StatusCode, "status code")
		var checkPb pb.FeedMessage
		if err := proto.Unmarshal(rr.Body.Bytes(), &checkPb); err != nil {
			t.Fatal(err)
		} else {
			assert.Greater(t, len(checkPb.Entity), 0, "should have entities")
		}
	})
	t.Run("trip_updates ok json", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/BA~rt/download_latest_rt/trip_updates.json", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)
		assert.Equal(t, "application/json", rr.Header().Get("content-type"), "content-type")
		assert.Equal(t, 200, rr.Result().StatusCode, "status code")
		var checkJson map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &checkJson); err != nil {
			t.Fatal(err)
		}
		if v, ok := checkJson["entity"].([]any); ok {
			assert.Greater(t, len(v), 0, "should have entities")
		} else {
			t.Fatal("expected entities")
		}
	})
	t.Run("trip_updates ok pb", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/BA~rt/download_latest_rt/trip_updates.pb", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)
		assert.Equal(t, "application/octet-stream", rr.Header().Get("content-type"), "content-type")
		assert.Equal(t, 200, rr.Result().StatusCode, "status code")
		var checkPb pb.FeedMessage
		if err := proto.Unmarshal(rr.Body.Bytes(), &checkPb); err != nil {
			t.Fatal(err)
		} else {
			assert.Greater(t, len(checkPb.Entity), 0, "should have entities")
		}
	})

	t.Run("geojson format only for vehicle_positions", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/BA~rt/download_latest_rt/alerts.geojson", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)
		assert.Equal(t, 400, rr.Result().StatusCode, "should return 400 for non-vehicle positions")
	})
	t.Run("feed not found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/asdxyz/download_latest_rt/alerts.json", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)
		assert.Equal(t, "application/json", rr.Header().Get("content-type"), "content-type")
		assert.Equal(t, 404, rr.Result().StatusCode, "status code")
	})
	t.Run("message not found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/BA~rt/download_latest_rt/asd.json", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)
		assert.Equal(t, "application/json", rr.Header().Get("content-type"), "content-type")
		assert.Equal(t, 404, rr.Result().StatusCode, "status code")
	})
}

func TestFeedDownloadRtVehiclePositions(t *testing.T) {
	_, restSrv, _ := testHandlersWithOptions(t, testconfig.Options{
		Storage: testdata.Path("server", "tmp"),
		RTJsons: []testconfig.RTJsonFile{
			{Feed: "CT~rt", Ftype: "realtime_vehicle_positions", Fname: "ct-vehicle-positions.pb.json"},
		},
	})

	t.Run("vehicle_positions geojson with data", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/CT~rt/download_latest_rt/vehicle_positions.geojson", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)
		assert.Equal(t, "application/geo+json", rr.Header().Get("content-type"), "content-type")
		assert.Equal(t, 200, rr.Result().StatusCode, "status code")

		var checkJson map[string]any
		if err := json.Unmarshal(rr.Body.Bytes(), &checkJson); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, "FeatureCollection", checkJson["type"], "should be a FeatureCollection")
		if features, ok := checkJson["features"].([]any); ok {
			assert.Greater(t, len(features), 0, "should have features")

			// Verify first feature structure
			if len(features) > 0 {
				feature := features[0].(map[string]any)
				assert.Equal(t, "Feature", feature["type"], "should be a Feature")

				geometry, ok := feature["geometry"].(map[string]any)
				assert.True(t, ok, "geometry should be present")
				assert.Equal(t, "Point", geometry["type"], "should be Point geometry")

				coordinates, ok := geometry["coordinates"].([]any)
				assert.True(t, ok, "coordinates should be present")
				assert.Equal(t, 2, len(coordinates), "should have 2 coordinates (lon, lat)")

				properties, ok := feature["properties"].(map[string]any)
				assert.True(t, ok, "properties should be present")
				assert.Contains(t, properties, "id", "should have id property")
				assert.Contains(t, properties, "latitude", "should have latitude property")
				assert.Contains(t, properties, "longitude", "should have longitude property")
			}
		} else {
			t.Fatal("expected features array")
		}
	})

	t.Run("vehicle_positions geojsonl with data", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/CT~rt/download_latest_rt/vehicle_positions.geojsonl", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)
		assert.Equal(t, "application/geo+json-seq", rr.Header().Get("content-type"), "content-type")
		assert.Equal(t, 200, rr.Result().StatusCode, "status code")

		body := rr.Body.Bytes()
		assert.Greater(t, len(body), 0, "should have content")

		// Verify streaming output
		lines := strings.Split(strings.TrimSpace(string(body)), "\n")
		assert.Greater(t, len(lines), 0, "should have lines")

		featureCount := 0
		for _, line := range lines {
			if line != "" {
				var feature map[string]any
				if err := json.Unmarshal([]byte(line), &feature); err != nil {
					t.Fatalf("invalid JSON in line: %s", line)
				}
				assert.Equal(t, "Feature", feature["type"], "should be a Feature")

				// Verify feature structure
				geometry, ok := feature["geometry"].(map[string]any)
				assert.True(t, ok, "geometry should be present")
				assert.Equal(t, "Point", geometry["type"], "should be Point geometry")

				coordinates, ok := geometry["coordinates"].([]any)
				assert.True(t, ok, "coordinates should be present")
				assert.Equal(t, 2, len(coordinates), "should have 2 coordinates (lon, lat)")

				properties, ok := feature["properties"].(map[string]any)
				assert.True(t, ok, "properties should be present")
				assert.Contains(t, properties, "id", "should have id property")
				assert.Contains(t, properties, "latitude", "should have latitude property")
				assert.Contains(t, properties, "longitude", "should have longitude property")

				featureCount++
			}
		}
		assert.Greater(t, featureCount, 0, "should have at least one feature")
	})

}

var _ meters.MeterProvider = (*fakeMeterProvider)(nil)

// fakeMeterProvider answers Check from a per-meter allow map, defaulting to
// allowed, and records every check and event so a test can assert what the
// handler asked about.
type fakeMeterProvider struct {
	allow  map[string]bool
	checks []fakeMeterCheck
	events []meters.MeterEvent
}

type fakeMeterCheck struct {
	name  string
	value float64
	dims  meters.Dimensions
}

func (p *fakeMeterProvider) NewMeter(meters.MeterUser) meters.Meterer {
	return &fakeMeter{provider: p}
}

func (p *fakeMeterProvider) Close() error { return nil }

func (p *fakeMeterProvider) Flush() error { return nil }

func (p *fakeMeterProvider) lastCheck(meterName string) (fakeMeterCheck, bool) {
	for i := len(p.checks) - 1; i >= 0; i-- {
		if p.checks[i].name == meterName {
			return p.checks[i], true
		}
	}
	return fakeMeterCheck{}, false
}

func (p *fakeMeterProvider) lastEvent(meterName string) meters.MeterEvent {
	for i := len(p.events) - 1; i >= 0; i-- {
		if p.events[i].Name == meterName {
			return p.events[i]
		}
	}
	return meters.MeterEvent{}
}

func (p *fakeMeterProvider) eventCount(meterName string) int {
	count := 0
	for _, event := range p.events {
		if event.Name == meterName {
			count++
		}
	}
	return count
}

type fakeMeter struct {
	provider *fakeMeterProvider
}

func (m *fakeMeter) Meter(_ context.Context, event meters.MeterEvent) error {
	m.provider.events = append(m.provider.events, event)
	return nil
}

func (m *fakeMeter) ApplyDimension(_ string, _ string) {}

func (m *fakeMeter) GetValue(_ context.Context, _ string, _ time.Time, _ time.Time, _ meters.Dimensions) (float64, bool) {
	return 0, false
}

func (m *fakeMeter) Check(_ context.Context, meterName string, value float64, dims meters.Dimensions) (bool, error) {
	m.provider.checks = append(m.provider.checks, fakeMeterCheck{name: meterName, value: value, dims: dims})
	if allow, ok := m.provider.allow[meterName]; ok {
		return allow, nil
	}
	return true, nil
}

// dimValue returns the value of one dimension, or "" if it is not present.
func dimValue(dims meters.Dimensions, key string) string {
	for _, dim := range dims {
		if dim.Key == key {
			return dim.Value
		}
	}
	return ""
}
