package validator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
)

//////////// helpers /////////////

// exampleReader returns an overlay feed reader.
func exampleReader(basepath string, overlaypath string) *tlcsv.Reader {
	reader, err := tlcsv.NewReader(".")
	if err != nil {
		return nil
	}
	reader.Adapter = tlcsv.NewOverlayAdapter(overlaypath, basepath)
	return reader
}

// testErrorHandler verifies that every error found is in the specified list.
type testErrorHandler struct {
	t                  *testing.T
	expectSourceErrors map[string][]testutil.ExpectError
	expectErrorCount   int
}

func (cr *testErrorHandler) HandleSourceErrors(fn string, errs []error, warns []error) {
	errs = append(errs, warns...)
	expecterrs := cr.expectSourceErrors[fn]
	cr.expectErrorCount += len(expecterrs)
	testutil.CheckErrors(expecterrs, errs, cr.t)
}

func (cr *testErrorHandler) HandleEntityErrors(ent tt.Entity, errs []error, warns []error) {
}

func (cr *testErrorHandler) AfterWrite(eid string, ent tt.Entity, emap *tt.EntityMap) error {
	var errs []error
	errs = append(errs, tt.CheckErrors(ent)...)
	errs = append(errs, tt.CheckWarnings(ent)...)
	expecterrs := testutil.GetExpectErrors(ent)
	cr.expectErrorCount += len(expecterrs)
	testutil.CheckErrors(expecterrs, errs, cr.t)
	return nil
}

//////////////

func TestValidator_Validate(t *testing.T) {
	basepath := testpath.RelPath("testdata/validator")
	searchpath := testpath.RelPath("testdata/validator/errors")
	files, err := os.ReadDir(searchpath)
	if err != nil {
		t.Error(err)
	}
	ctx := context.TODO()
	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		t.Run(file.Name(), func(t *testing.T) {
			reader := exampleReader(basepath, filepath.Join(searchpath, file.Name()))
			handler := testErrorHandler{
				t:                  t,
				expectSourceErrors: map[string][]testutil.ExpectError{},
			}
			// Directly read the expect_errors.txt
			reader.Adapter.ReadRows("expect_errors.txt", func(row tlcsv.Row) {
				fn := func(a string, _ bool) string { return a }
				ee := testutil.NewExpectError(
					fn(row.Get("filename")),
					fn(row.Get("entity_id")),
					fn(row.Get("field")),
					fn(row.Get("error")),
				)
				handler.expectSourceErrors[ee.Filename] = append(handler.expectSourceErrors[ee.Filename], ee)
			})
			////////
			// For every overlay feed, check that every error is expected
			// At least one error must be specified per overlay feed, otherwise fail
			opts := Options{}
			opts.ErrorHandler = &handler
			v, _ := NewValidator(reader, opts)
			v.AddExtension(&handler)
			v.Validate(ctx)
			if handler.expectErrorCount == 0 {
				t.Errorf("feed did not contain any test cases")
			}
		})
	}
}

func TestValidator_BestPractices(t *testing.T) {
	// TODO: Combine with above... test best practice rules.
	ctx := context.TODO()
	basepath := testpath.RelPath("testdata/validator")
	searchpath := testpath.RelPath("testdata/validator/best-practices")
	files, err := os.ReadDir(searchpath)
	if err != nil {
		t.Error(err)
	}
	for _, file := range files {
		if !file.IsDir() {
			continue
		}
		t.Run(file.Name(), func(t *testing.T) {
			reader := exampleReader(basepath, filepath.Join(searchpath, file.Name()))
			handler := testErrorHandler{
				t:                  t,
				expectSourceErrors: map[string][]testutil.ExpectError{},
			}
			// Directly read the expect_errors.txt
			reader.Adapter.ReadRows("expect_errors.txt", func(row tlcsv.Row) {
				fn := func(a string, _ bool) string { return a }
				ee := testutil.NewExpectError(
					fn(row.Get("filename")),
					fn(row.Get("entity_id")),
					fn(row.Get("field")),
					fn(row.Get("error")),
				)
				handler.expectSourceErrors[ee.Filename] = append(handler.expectSourceErrors[ee.Filename], ee)
			})
			////////
			// For every overlay feed, check that every error is expected
			// At least one error must be specified per overlay feed, otherwise fail
			opts := Options{}
			opts.ErrorHandler = &handler
			opts.BestPractices = true
			v, _ := NewValidator(reader, opts)
			v.AddExtension(&handler)
			result, _ := v.Validate(ctx)
			_ = result
			if handler.expectErrorCount == 0 {
				t.Errorf("feed did not contain any test cases")
			}
		})
	}
}

func TestSaveValidationReport(t *testing.T) {
	ctx := context.TODO()
	reader, err := tlcsv.NewReader(testpath.RelPath("testdata/rt/ct.zip"))
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := os.ReadFile(testpath.RelPath(filepath.Join("testdata/rt", r.URL.Path)))
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))

	tz, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Date(2023, 11, 7, 17, 05, 0, 0, tz)
	opts := Options{
		IncludeRealtimeJson: true,
		EvaluateAt:          now,
		ValidateRealtimeMessages: []string{
			ts.URL + "/ct-trip-stats.json",
			ts.URL + "/ct-vehicle-stats.json",
		},
	}

	v, _ := NewValidator(reader, opts)
	result, err := v.Validate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	testdb.TempSqlite(func(atx tldb.Adapter) error {
		if err := SaveValidationReport(ctx, atx, result, 1, ""); err != nil {
			t.Fatal(err)
		}
		return nil
	})
}
