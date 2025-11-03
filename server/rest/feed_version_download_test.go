package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestFeedVersionDownloadRequest(t *testing.T) {
	_, restSrv, cfg := testHandlersWithOptions(t, testconfig.Options{
		Storage: testdata.Path("server", "tmp"),
	})

	t.Run("ok", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feed_versions/d2813c293bcfd7a97dde599527ae6c62c98e66c6/download", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.UseDefaultUserMiddleware("test", cfg.Roles.AdminRole)(restSrv)
		asAdmin.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 200 {
			t.Errorf("got status code %d, expected 200", sc)
		}
		if sc := len(rr.Body.Bytes()); sc != 59324 {
			t.Errorf("got %d bytes, expected 59324", sc)
		}
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
		asUser := usercheck.UseDefaultUserMiddleware("testuser")(restSrv)
		asUser.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 401 {
			t.Errorf("got status code %d, expected 401", sc)
		}
	})
	t.Run("not authorized as user, only current download role", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feed_versions/d2813c293bcfd7a97dde599527ae6c62c98e66c6/download", nil)
		rr := httptest.NewRecorder()
		asUser := usercheck.UseDefaultUserMiddleware("testuser", cfg.Roles.DownloadCurrentFeedVersionRole)(restSrv)
		asUser.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 401 {
			t.Errorf("got status code %d, expected 401", sc)
		}
	})
	t.Run("authorized as user", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feed_versions/d2813c293bcfd7a97dde599527ae6c62c98e66c6/download", nil)
		rr := httptest.NewRecorder()
		asUser := usercheck.UseDefaultUserMiddleware("testuser", cfg.Roles.DownloadHistoricFeedVersionRole)(restSrv)
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
		asUser := usercheck.UseDefaultUserMiddleware("testuser", cfg.Roles.DownloadHistoricFeedVersionRole)(restSrv)
		asUser.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 401 {
			t.Errorf("got status code %d, expected 401", sc)
		}
	})
	t.Run("not found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feed_versions/asdxyz/download", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.UseDefaultUserMiddleware("test", cfg.Roles.AdminRole)(restSrv)
		asAdmin.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 404 {
			t.Errorf("got status code %d, expected 404", sc)
		}
	})
}

func TestFeedDownloadLatestRequest(t *testing.T) {
	_, restSrv, cfg := testHandlersWithOptions(t, testconfig.Options{
		Storage: testdata.Path("server", "tmp"),
	})

	t.Run("ok", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/CT/download_latest_feed_version", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.UseDefaultUserMiddleware("test", cfg.Roles.AdminRole)(restSrv)
		asAdmin.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 200 {
			t.Errorf("got status code %d, expected 200", sc)
		}
		if sc := len(rr.Body.Bytes()); sc != 59324 {
			t.Errorf("got %d bytes, expected 59324", sc)
		}
	})
	t.Run("ok as user", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/CT/download_latest_feed_version", nil)
		rr := httptest.NewRecorder()
		asUser := usercheck.UseDefaultUserMiddleware("testuser", cfg.Roles.DownloadCurrentFeedVersionRole)(restSrv)
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
		asUser := usercheck.UseDefaultUserMiddleware("testuser")(restSrv)
		asUser.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 401 {
			t.Errorf("got status code %d, expected 401", sc)
		}
	})
	t.Run("not authorized as user, not redistributable", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/BA/download_latest_feed_version", nil)
		rr := httptest.NewRecorder()
		asUser := usercheck.UseDefaultUserMiddleware("testuser", cfg.Roles.DownloadCurrentFeedVersionRole)(restSrv)
		asUser.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 401 {
			t.Errorf("got status code %d, expected 401", sc)
		}
	})

	t.Run("not found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/asdxyz/download_latest_feed_version", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.UseDefaultUserMiddleware("test", cfg.Roles.AdminRole)(restSrv)
		asAdmin.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 404 {
			t.Errorf("got status code %d, expected 404", sc)
		}
	})
}

func TestFeedDownloadRtLatestRequest(t *testing.T) {
	_, restSrv, cfg := testHandlersWithOptions(t, testconfig.Options{
		Storage: testdata.Path("server", "tmp"),
		RTJsons: []testconfig.RTJsonFile{
			{Feed: "BA~rt", Ftype: "realtime_alerts", Fname: "BA-alerts.json"},
			{Feed: "BA~rt", Ftype: "realtime_trip_updates", Fname: "BA.json"},
		},
	})
	t.Run("ok as user", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/BA~rt/download_latest_rt/alerts.json", nil)
		rr := httptest.NewRecorder()
		asUser := usercheck.UseDefaultUserMiddleware("testuser", cfg.Roles.DownloadHistoricFeedVersionRole)(restSrv)
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
		asAdmin := usercheck.UseDefaultUserMiddleware("test", cfg.Roles.AdminRole)(restSrv)
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
		asAdmin := usercheck.UseDefaultUserMiddleware("test", cfg.Roles.AdminRole)(restSrv)
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
		asAdmin := usercheck.UseDefaultUserMiddleware("test", cfg.Roles.AdminRole)(restSrv)
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
		asAdmin := usercheck.UseDefaultUserMiddleware("test", cfg.Roles.AdminRole)(restSrv)
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
		asAdmin := usercheck.UseDefaultUserMiddleware("test", cfg.Roles.AdminRole)(restSrv)
		asAdmin.ServeHTTP(rr, req)
		assert.Equal(t, 400, rr.Result().StatusCode, "should return 400 for non-vehicle positions")
	})
	t.Run("feed not found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/asdxyz/download_latest_rt/alerts.json", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.UseDefaultUserMiddleware("test", cfg.Roles.AdminRole)(restSrv)
		asAdmin.ServeHTTP(rr, req)
		assert.Equal(t, "application/json", rr.Header().Get("content-type"), "content-type")
		assert.Equal(t, 404, rr.Result().StatusCode, "status code")
	})
	t.Run("message not found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/BA~rt/download_latest_rt/asd.json", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.UseDefaultUserMiddleware("test", cfg.Roles.AdminRole)(restSrv)
		asAdmin.ServeHTTP(rr, req)
		assert.Equal(t, "application/json", rr.Header().Get("content-type"), "content-type")
		assert.Equal(t, 404, rr.Result().StatusCode, "status code")
	})
}

func TestFeedDownloadRtVehiclePositions(t *testing.T) {
	_, restSrv, cfg := testHandlersWithOptions(t, testconfig.Options{
		Storage: testdata.Path("server", "tmp"),
		RTJsons: []testconfig.RTJsonFile{
			{Feed: "CT~rt", Ftype: "realtime_vehicle_positions", Fname: "ct-vehicle-positions.pb.json"},
		},
	})

	t.Run("vehicle_positions geojson with data", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/CT~rt/download_latest_rt/vehicle_positions.geojson", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.UseDefaultUserMiddleware("test", cfg.Roles.AdminRole)(restSrv)
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
		asAdmin := usercheck.UseDefaultUserMiddleware("test", cfg.Roles.AdminRole)(restSrv)
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
