package rest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/interline-io/transitland-mw/auth/authn"
	"github.com/interline-io/transitland-mw/auth/mw/usercheck"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestFeedVersionDownloadRequest(t *testing.T) {
	_, restSrv, _ := testHandlersWithOptions(t, testconfig.Options{
		Storage: testdata.Path("tmp"),
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
	_, restSrv, _ := testHandlersWithOptions(t, testconfig.Options{
		Storage: testdata.Path("tmp"),
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

func TestFeedDownloadRtLatestRequest(t *testing.T) {
	_, restSrv, _ := testHandlersWithOptions(t, testconfig.Options{
		Storage: testdata.Path("tmp"),
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
			assert.Equal(t, 8, len(v), "entity count")
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
			assert.Equal(t, 8, len(checkPb.Entity), "entity count")
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
			assert.Equal(t, 48, len(v), "entity count")
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
			assert.Equal(t, 48, len(checkPb.Entity), "entity count")
		}
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
