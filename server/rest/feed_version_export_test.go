package rest

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/stretchr/testify/assert"
)

func TestFeedVersionExportRequest(t *testing.T) {
	_, restSrv, _ := testHandlersWithOptions(t, testconfig.Options{
		Storage: testdata.Path("server", "tmp"),
	})

	t.Run("basic export single feed version", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/feed_versions/export", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)

		assert.Equal(t, 200, rr.Result().StatusCode, "status code")
		assert.Equal(t, "application/zip", rr.Header().Get("Content-Type"), "content-type")
		assert.Contains(t, rr.Header().Get("Content-Disposition"), "attachment", "content-disposition")
		assert.Contains(t, rr.Header().Get("Content-Disposition"), ".zip", "zip filename")

		// Verify it's a valid ZIP file
		zipReader, err := zip.NewReader(bytes.NewReader(rr.Body.Bytes()), int64(rr.Body.Len()))
		assert.NoError(t, err, "should be valid zip")
		assert.Greater(t, len(zipReader.File), 0, "should have files in zip")

		// Check for expected GTFS files
		fileNames := make(map[string]bool)
		for _, f := range zipReader.File {
			fileNames[f.Name] = true
		}
		assert.True(t, fileNames["agency.txt"], "should have agency.txt")
		assert.True(t, fileNames["stops.txt"], "should have stops.txt")
		assert.True(t, fileNames["routes.txt"], "should have routes.txt")
	})

	t.Run("export multiple feed versions (merge)", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{
				"d2813c293bcfd7a97dde599527ae6c62c98e66c6",
				"e535eb2b3b9ac3ef15d82c56575e914575e732e0",
			},
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/feed_versions/export", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)

		assert.Equal(t, 200, rr.Result().StatusCode, "status code")
		assert.Equal(t, "application/zip", rr.Header().Get("Content-Type"), "content-type")

		// Verify merged ZIP
		zipReader, err := zip.NewReader(bytes.NewReader(rr.Body.Bytes()), int64(rr.Body.Len()))
		assert.NoError(t, err, "should be valid zip")
		assert.Greater(t, len(zipReader.File), 0, "should have files in merged zip")
	})

	t.Run("export with transformations", func(t *testing.T) {
		simplifyShapes := 10.0
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
			Transforms: &ExportTransforms{
				Prefix:             "test_",
				NormalizeTimezones: true,
				SimplifyShapes:     &simplifyShapes,
				UseBasicRouteTypes: true,
			},
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/feed_versions/export", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)

		assert.Equal(t, 200, rr.Result().StatusCode, "status code")
		assert.Equal(t, "application/zip", rr.Header().Get("Content-Type"), "content-type")
	})

	t.Run("not authorized as anon", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/feed_versions/export", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		restSrv.ServeHTTP(rr, req)

		assert.Equal(t, 401, rr.Result().StatusCode, "should be unauthorized")
	})

	t.Run("not authorized as user, missing role", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/feed_versions/export", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		asUser := usercheck.NewUserDefaultMiddleware(func() authn.User {
			return authn.NewCtxUser("testuser", "", "").WithRoles("some_other_role")
		})(restSrv)
		asUser.ServeHTTP(rr, req)

		assert.Equal(t, 401, rr.Result().StatusCode, "should be unauthorized without export role")
	})

	t.Run("not authorized as user, only download role", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/feed_versions/export", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		asUser := usercheck.NewUserDefaultMiddleware(func() authn.User {
			return authn.NewCtxUser("testuser", "", "").WithRoles("tl_download_fv_historic")
		})(restSrv)
		asUser.ServeHTTP(rr, req)

		assert.Equal(t, 401, rr.Result().StatusCode, "should be unauthorized with only download role")
	})

	t.Run("authorized as user with export role", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/feed_versions/export", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		asUser := usercheck.NewUserDefaultMiddleware(func() authn.User {
			return authn.NewCtxUser("testuser", "", "").WithRoles("tl_export_feed_versions")
		})(restSrv)
		asUser.ServeHTTP(rr, req)

		assert.Equal(t, 200, rr.Result().StatusCode, "should be authorized with export role")
	})

	t.Run("bad request - empty feed_version_keys", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{},
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/feed_versions/export", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)

		assert.Equal(t, 400, rr.Result().StatusCode, "should be bad request")
		assert.Contains(t, rr.Body.String(), "feed_version_keys is required", "error message")
	})

	t.Run("bad request - invalid JSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/feed_versions/export", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)

		assert.Equal(t, 400, rr.Result().StatusCode, "should be bad request")
	})

	t.Run("not found - feed version does not exist", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{"nonexistent_sha1"},
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/feed_versions/export", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)

		assert.Equal(t, 404, rr.Result().StatusCode, "should be not found")
		assert.Contains(t, rr.Body.String(), "not found", "error message")
	})

	t.Run("forbidden - feed version does not allow redistribution", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907"}, // BA feed - no redistribution
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/feed_versions/export", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)

		assert.Equal(t, 403, rr.Result().StatusCode, "should be forbidden")
		assert.Contains(t, rr.Body.String(), "does not allow redistribution", "error message")
	})

	t.Run("bad request - feed version not imported", func(t *testing.T) {
		// This test would need a feed version that exists but hasn't been imported
		// The test database may not have such a case, so this is a placeholder
		// In a real scenario, you'd set up a feed version without import
		t.Skip("requires test data with unimported feed version")
	})

	t.Run("export by feed version ID", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{"1"}, // Using ID instead of SHA1
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/feed_versions/export", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)

		assert.Equal(t, 200, rr.Result().StatusCode, "should work with ID")
		assert.Equal(t, "application/zip", rr.Header().Get("Content-Type"), "content-type")
	})

	t.Run("export mixed IDs and SHA1s", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{
				"1", // ID
				"e535eb2b3b9ac3ef15d82c56575e914575e732e0", // SHA1
			},
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/feed_versions/export", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)

		assert.Equal(t, 200, rr.Result().StatusCode, "should work with mixed IDs and SHA1s")
	})

	t.Run("method not allowed - GET request", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feed_versions/export", nil)
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)

		assert.Equal(t, 405, rr.Result().StatusCode, "should be method not allowed for GET")
	})

	t.Run("verify ZIP contents structure", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		}
		body, _ := json.Marshal(reqBody)
		req, _ := http.NewRequest("POST", "/feed_versions/export", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
		asAdmin.ServeHTTP(rr, req)

		assert.Equal(t, 200, rr.Result().StatusCode)

		// Parse ZIP and verify structure
		zipReader, err := zip.NewReader(bytes.NewReader(rr.Body.Bytes()), int64(rr.Body.Len()))
		assert.NoError(t, err)

		// Read agency.txt to verify it has content
		for _, f := range zipReader.File {
			if f.Name == "agency.txt" {
				rc, err := f.Open()
				assert.NoError(t, err)
				defer rc.Close()

				content, err := io.ReadAll(rc)
				assert.NoError(t, err)
				assert.Greater(t, len(content), 0, "agency.txt should have content")

				// Verify it's CSV format (has header)
				lines := strings.Split(string(content), "\n")
				assert.Greater(t, len(lines), 1, "should have header and data rows")
				assert.Contains(t, lines[0], "agency_id", "should have CSV header")
				break
			}
		}
	})
}
