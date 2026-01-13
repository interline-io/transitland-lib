package rest

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	sq "github.com/irees/squirrel"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/stretchr/testify/assert"
)

func TestFeedVersionExportRequest(t *testing.T) {
	_, restSrv, cfg := testHandlersWithOptions(t, testconfig.Options{
		Storage: testdata.Path("server", "tmp"),
	})

	// Get integer IDs for some feed versions
	ctx := context.Background()
	type fvQuery struct {
		Sha1 string
		ID   int
	}
	var fvs []fvQuery
	if err := dbutil.Select(ctx, cfg.Finder.DBX(), sq.StatementBuilder.Select("id", "sha1").From("feed_versions"), &fvs); err != nil {
		t.Fatalf("failed to query feed versions: %v", err)
	}
	fvidBySha1 := map[string]int{}
	for _, fv := range fvs {
		fvidBySha1[fv.Sha1] = fv.ID
	}
	caltrainFv := "d2813c293bcfd7a97dde599527ae6c62c98e66c6"
	hartFv := "c969427f56d3a645195dd8365cde6d7feae7e99b"
	bartFv := "e535eb2b3b9ac3ef15d82c56575e914575e732e0" // no redistribution

	// Common middleware setups
	asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
	asUserWithoutRole := usercheck.NewUserDefaultMiddleware(func() authn.User {
		return authn.NewCtxUser("testuser", "", "").WithRoles("some_other_role")
	})(restSrv)
	asUserWithDownloadRole := usercheck.NewUserDefaultMiddleware(func() authn.User {
		return authn.NewCtxUser("testuser", "", "").WithRoles("tl_download_fv_historic")
	})(restSrv)
	asUserWithExportRole := usercheck.NewUserDefaultMiddleware(func() authn.User {
		return authn.NewCtxUser("testuser", "", "").WithRoles("tl_export_feed_versions")
	})(restSrv)

	t.Run("basic export single feed version", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{caltrainFv},
		}
		rr := makeExportRequest(t, reqBody, asAdmin)

		assert.Equal(t, 200, rr.Result().StatusCode, "status code")
		validateZipResponse(t, rr, map[string]int{
			"agency.txt":          1,
			"calendar.txt":        27,
			"calendar_dates.txt":  36,
			"fare_attributes.txt": 6,
			"fare_rules.txt":      216,
			"routes.txt":          6,
			"shapes.txt":          3008,
			"stop_times.txt":      2853,
			"stops.txt":           64,
			"trips.txt":           185,
		})
		if err := makeTempReader(t, rr.Body.Bytes(), func(t *testing.T, reader *tlcsv.Reader) {
			var entIds []string
			for ent := range reader.Routes() {
				entIds = append(entIds, ent.RouteID.Val)
			}
			assert.Equal(
				t,
				[]string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
				entIds,
				"route IDs",
			)
		}); err != nil {
			t.Fatalf("test failed: %v", err)
		}
	})

	t.Run("export multiple feed versions (merge)", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{caltrainFv, hartFv},
		}
		rr := makeExportRequest(t, reqBody, asAdmin)

		assert.Equal(t, 200, rr.Result().StatusCode, "status code")
		validateZipResponse(t, rr, map[string]int{
			"agency.txt":          2,
			"calendar.txt":        44,
			"calendar_dates.txt":  45,
			"fare_attributes.txt": 12,
			"fare_rules.txt":      258,
			"routes.txt":          51,
			"shapes.txt":          59160,
			"stop_times.txt":      438360,
			"stops.txt":           2413,
			"trips.txt":           14903,
		})
		if err := makeTempReader(t, rr.Body.Bytes(), func(t *testing.T, reader *tlcsv.Reader) {
			var entIds []string
			for ent := range reader.Routes() {
				entIds = append(entIds, ent.RouteID.Val)
			}
			assert.Equal(
				t,
				[]string{"1", "12", "14", "15", "16", "17", "19", "20", "24", "25", "275", "30", "31", "32", "33", "34", "35", "36", "360", "37", "38", "39", "400", "42", "45", "46", "48", "5", "51", "6", "60", "7", "75", "8", "9", "96", "97", "570", "571", "572", "573", "574", "800", "PWT", "SKY", "Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
				entIds,
				"route IDs",
			)
		}); err != nil {
			t.Fatalf("test failed: %v", err)
		}
	})

	t.Run("export with transformations", func(t *testing.T) {
		simplifyShapes := 10.0
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{caltrainFv},
			Transforms: &ExportTransforms{
				Prefix:             "test_",
				NormalizeTimezones: true,
				SimplifyShapes:     &simplifyShapes,
				UseBasicRouteTypes: true,
			},
		}
		rr := makeExportRequest(t, reqBody, asAdmin)

		assert.Equal(t, 200, rr.Result().StatusCode, "status code")
		validateZipResponse(t, rr, map[string]int{
			"agency.txt":          1,
			"calendar.txt":        27,
			"calendar_dates.txt":  36,
			"fare_attributes.txt": 6,
			"fare_rules.txt":      216,
			"routes.txt":          6,
			"shapes.txt":          2046,
			"trips.txt":           185,
		})
		if err := makeTempReader(t, rr.Body.Bytes(), func(t *testing.T, reader *tlcsv.Reader) {
			var entIds []string
			for ent := range reader.Routes() {
				entIds = append(entIds, ent.RouteID.Val)
			}
			assert.Equal(
				t,
				[]string{"test_Bu-130", "test_Li-130", "test_Lo-130", "test_TaSj-130", "test_Gi-130", "test_Sp-130"},
				entIds,
				"route IDs",
			)
		}); err != nil {
			t.Fatalf("test failed: %v", err)
		}
	})

	t.Run("export with lexicographic sort ascending", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{caltrainFv},
			Transforms: &ExportTransforms{
				LexicographicSort: "asc",
			},
		}
		rr := makeExportRequest(t, reqBody, asAdmin)

		assert.Equal(t, 200, rr.Result().StatusCode, "status code")
		validateZipResponse(t, rr, map[string]int{
			"agency.txt":          1,
			"calendar.txt":        27,
			"calendar_dates.txt":  36,
			"fare_attributes.txt": 6,
			"fare_rules.txt":      216,
			"routes.txt":          6,
			"shapes.txt":          3008,
			"stop_times.txt":      2853,
			"stops.txt":           64,
			"trips.txt":           185,
		})
		// Verify stops are sorted lexicographically (ascending)
		if err := makeTempReader(t, rr.Body.Bytes(), func(t *testing.T, reader *tlcsv.Reader) {
			var stopIds []string
			for ent := range reader.Stops() {
				stopIds = append(stopIds, ent.StopID.Val)
			}
			// Check that stops are sorted
			for i := 1; i < len(stopIds); i++ {
				if stopIds[i-1] > stopIds[i] {
					t.Errorf("stops not sorted ascending: %s > %s", stopIds[i-1], stopIds[i])
				}
			}
		}); err != nil {
			t.Fatalf("test failed: %v", err)
		}
	})

	t.Run("export with lexicographic sort descending", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{caltrainFv},
			Transforms: &ExportTransforms{
				LexicographicSort: "desc",
			},
		}
		rr := makeExportRequest(t, reqBody, asAdmin)

		assert.Equal(t, 200, rr.Result().StatusCode, "status code")
		validateZipResponse(t, rr, map[string]int{
			"agency.txt":          1,
			"calendar.txt":        27,
			"calendar_dates.txt":  36,
			"fare_attributes.txt": 6,
			"fare_rules.txt":      216,
			"routes.txt":          6,
			"shapes.txt":          3008,
			"stop_times.txt":      2853,
			"stops.txt":           64,
			"trips.txt":           185,
		})
		// Verify stops are sorted lexicographically (descending)
		if err := makeTempReader(t, rr.Body.Bytes(), func(t *testing.T, reader *tlcsv.Reader) {
			var stopIds []string
			for ent := range reader.Stops() {
				stopIds = append(stopIds, ent.StopID.Val)
			}
			// Check that stops are sorted descending
			for i := 1; i < len(stopIds); i++ {
				if stopIds[i-1] < stopIds[i] {
					t.Errorf("stops not sorted descending: %s < %s", stopIds[i-1], stopIds[i])
				}
			}
		}); err != nil {
			t.Fatalf("test failed: %v", err)
		}
	})

	t.Run("export by feed version ID", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{fmt.Sprintf("%d", fvidBySha1[caltrainFv])}, // Using ID instead of SHA1
		}
		rr := makeExportRequest(t, reqBody, asAdmin)

		assert.Equal(t, 200, rr.Result().StatusCode, "should work with ID")
		validateZipResponse(t, rr, map[string]int{
			"agency.txt":          1,
			"calendar.txt":        27,
			"calendar_dates.txt":  36,
			"fare_attributes.txt": 6,
			"fare_rules.txt":      216,
			"routes.txt":          6,
			"shapes.txt":          3008,
			"stop_times.txt":      2853,
			"stops.txt":           64,
			"trips.txt":           185,
		})
		if err := makeTempReader(t, rr.Body.Bytes(), func(t *testing.T, reader *tlcsv.Reader) {
			var entIds []string
			for ent := range reader.Routes() {
				entIds = append(entIds, ent.RouteID.Val)
			}
			assert.Equal(
				t,
				[]string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
				entIds,
				"route IDs",
			)
		}); err != nil {
			t.Fatalf("test failed: %v", err)
		}
	})

	t.Run("export mixed IDs and SHA1s", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{
				fmt.Sprintf("%d", fvidBySha1[caltrainFv]), // ID
				hartFv, // SHA1
			},
		}
		rr := makeExportRequest(t, reqBody, asAdmin)

		assert.Equal(t, 200, rr.Result().StatusCode, "should work with mixed IDs and SHA1s")
		validateZipResponse(t, rr, map[string]int{
			"agency.txt":          2,
			"calendar.txt":        44,
			"calendar_dates.txt":  45,
			"fare_attributes.txt": 12,
			"fare_rules.txt":      258,
			"routes.txt":          51,
			"shapes.txt":          59160,
			"stop_times.txt":      438360,
			"stops.txt":           2413,
			"trips.txt":           14903,
		})
		if err := makeTempReader(t, rr.Body.Bytes(), func(t *testing.T, reader *tlcsv.Reader) {
			var entIds []string
			for ent := range reader.Agencies() {
				entIds = append(entIds, ent.AgencyID.Val)
			}
			assert.Equal(
				t,
				[]string{"", "caltrain-ca-us"},
				entIds,
				"agency IDs",
			)
		}); err != nil {
			t.Fatalf("test failed: %v", err)
		}
	})

	t.Run("not authorized as anon", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{caltrainFv},
		}
		rr := makeExportRequest(t, reqBody, restSrv)

		assert.Equal(t, 401, rr.Result().StatusCode, "should be unauthorized")
	})

	t.Run("not authorized as user, missing role", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{caltrainFv},
		}
		rr := makeExportRequest(t, reqBody, asUserWithoutRole)

		assert.Equal(t, 401, rr.Result().StatusCode, "should be unauthorized without export role")
	})

	t.Run("not authorized as user, only download role", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{caltrainFv},
		}
		rr := makeExportRequest(t, reqBody, asUserWithDownloadRole)

		assert.Equal(t, 401, rr.Result().StatusCode, "should be unauthorized with only download role")
	})

	t.Run("authorized as user with export role", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{caltrainFv},
		}
		rr := makeExportRequest(t, reqBody, asUserWithExportRole)

		assert.Equal(t, 200, rr.Result().StatusCode, "should be authorized with export role")
	})

	t.Run("bad request - empty feed_version_keys", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{},
		}
		rr := makeExportRequest(t, reqBody, asAdmin)

		assert.Equal(t, 400, rr.Result().StatusCode, "should be bad request")
		assert.Contains(t, rr.Body.String(), "feed_version_keys is required", "error message")
	})

	t.Run("bad request - invalid JSON", func(t *testing.T) {
		rr := makeExportRequestWithRawBody(t, strings.NewReader("invalid json"), asAdmin)

		assert.Equal(t, 400, rr.Result().StatusCode, "should be bad request")
	})

	t.Run("not found - feed version does not exist", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{"nonexistent_sha1"},
		}
		rr := makeExportRequest(t, reqBody, asAdmin)

		assert.Equal(t, 404, rr.Result().StatusCode, "should be not found")
		assert.Contains(t, rr.Body.String(), "not found", "error message")
	})

	t.Run("forbidden - feed version does not allow redistribution", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{bartFv}, // BA feed - no redistribution
		}
		rr := makeExportRequest(t, reqBody, asAdmin)

		assert.Equal(t, 403, rr.Result().StatusCode, "should be forbidden")
		assert.Contains(t, rr.Body.String(), "does not allow redistribution", "error message")
	})

	t.Run("bad request - invalid lexicographic_sort value", func(t *testing.T) {
		reqBody := FeedVersionExportRequest{
			FeedVersionKeys: []string{caltrainFv},
			Transforms: &ExportTransforms{
				LexicographicSort: "invalid",
			},
		}
		rr := makeExportRequest(t, reqBody, asAdmin)

		assert.Equal(t, 400, rr.Result().StatusCode, "should be bad request")
		assert.Contains(t, rr.Body.String(), "invalid lexicographic_sort", "error message")
	})

	t.Run("bad request - feed version not imported", func(t *testing.T) {
		// This test would need a feed version that exists but hasn't been imported
		// The test database may not have such a case, so this is a placeholder
		// In a real scenario, you'd set up a feed version without import
		t.Skip("requires test data with unimported feed version")
	})

	// t.Run("method not allowed - GET request", func(t *testing.T) {
	// 	req, _ := http.NewRequest("GET", "/feed_versions/export", nil)
	// 	rr := httptest.NewRecorder()
	// 	asAdmin := usercheck.AdminDefaultMiddleware("test")(restSrv)
	// 	asAdmin.ServeHTTP(rr, req)

	// 	assert.Equal(t, 405, rr.Result().StatusCode, "should be method not allowed for GET")
	// })

}

// Test helper functions
func makeExportRequest(t *testing.T, reqBody FeedVersionExportRequest, handler http.Handler) *httptest.ResponseRecorder {
	body, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("failed to marshal request body: %v", err)
	}

	req, err := http.NewRequest("POST", "/feed_versions/export", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func makeExportRequestWithRawBody(t *testing.T, body io.Reader, handler http.Handler) *httptest.ResponseRecorder {
	req, err := http.NewRequest("POST", "/feed_versions/export", body)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr
}

func validateZipResponse(t *testing.T, rr *httptest.ResponseRecorder, expectedFiles map[string]int) *zip.Reader {
	assert.Equal(t, "application/zip", rr.Header().Get("Content-Type"), "content-type")
	assert.Contains(t, rr.Header().Get("Content-Disposition"), "attachment", "content-disposition")
	assert.Contains(t, rr.Header().Get("Content-Disposition"), ".zip", "zip filename")

	// Verify it's a valid ZIP file
	zipReader, err := zip.NewReader(bytes.NewReader(rr.Body.Bytes()), int64(rr.Body.Len()))
	assert.NoError(t, err, "should be valid zip")
	assert.Greater(t, len(zipReader.File), 0, "should have files in zip")

	if len(expectedFiles) > 0 {
		// Get all files in ZIP for logging
		allFiles := make(map[string]int)
		for _, f := range zipReader.File {
			allFiles[f.Name] = countLinesInZipFile(t, zipReader, f.Name)
		}

		// Log all file line counts for debugging
		t.Logf("ZIP file contents and line counts:")
		for filename, lineCount := range allFiles {
			t.Logf("  %s: %d lines", filename, lineCount)
		}
		fmt.Printf("%#v\n", allFiles)

		// Check expected files and line counts
		for filename, expectedLines := range expectedFiles {
			actualLines, exists := allFiles[filename]
			assert.True(t, exists, "should have file %s", filename)

			if expectedLines == -1 {
				// Just check for presence (already done above)
				t.Logf("File %s exists with %d lines (presence check only)", filename, actualLines)
			} else {
				// Check specific line count
				assert.Equal(t, expectedLines, actualLines, "file %s should have %d lines (excluding header), got %d", filename, expectedLines, actualLines)
			}
		}
	}

	return zipReader
}

// countLinesInZipFile counts the number of data lines in a CSV file within a ZIP (excluding header)
func countLinesInZipFile(t *testing.T, zipReader *zip.Reader, filename string) int {
	for _, f := range zipReader.File {
		if f.Name == filename {
			rc, err := f.Open()
			assert.NoError(t, err, "should be able to open %s", filename)
			defer rc.Close()

			lineCount := 0
			// Use the tlcsv ReadRows pattern to count rows (excludes header automatically)
			err = tlcsv.ReadRows(rc, func(row tlcsv.Row) {
				lineCount++
			})
			assert.NoError(t, err, "should be able to read CSV rows from %s", filename)
			return lineCount
		}
	}
	t.Errorf("file %s not found in ZIP", filename)
	return 0
}

func makeTempReader(t *testing.T, data []byte, cb func(t *testing.T, reader *tlcsv.Reader)) error {
	tmpFile, err := os.CreateTemp("", "exported_feed_*.zip")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %v", err)
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("failed to write to temp file: %v", err)
	}

	reader, err := tlcsv.NewReader(tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to create tlcsv reader: %v", err)
	}
	defer reader.Close()

	cb(t, reader)

	return nil
}
