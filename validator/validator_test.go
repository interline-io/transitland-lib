package validator

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlcsv"
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
	// errs = append(errs, warns...)
	// expecterrs := cr.expectSourceErrors[fn]
	// cr.expectErrorCount += len(expecterrs)
	// testutil.CheckErrors(expecterrs, errs, cr.t)
}

func (cr *testErrorHandler) HandleEntityErrors(ent tl.Entity, errs []error, warns []error) {
}

type hasWarnings interface {
	Warnings() []error
}

func (cr *testErrorHandler) AfterWrite(eid string, ent tl.Entity, emap *tl.EntityMap) error {
	errs := ent.Errors()
	if v, ok := ent.(hasWarnings); ok {
		errs = append(errs, v.Warnings()...)
	}
	expecterrs := testutil.GetExpectErrors(ent)
	cr.expectErrorCount += len(expecterrs)
	testutil.CheckErrors(expecterrs, errs, cr.t)
	return nil
}

//////////////

func TestValidator_Validate(t *testing.T) {
	basepath := testutil.RelPath("test/data/validator")
	searchpath := testutil.RelPath("test/data/validator/errors")
	files, err := ioutil.ReadDir(searchpath)
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
				fn := func(a string, b bool) string { return a }
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
			opts.AllowEntityErrors = true
			opts.AllowReferenceErrors = true
			v, _ := NewValidator(reader, opts)
			v.AddExtension(&handler)
			v.Validate()
			if handler.expectErrorCount == 0 {
				t.Errorf("feed did not contain any test cases")
			}
		})
	}
}

func TestValidator_BestPractices(t *testing.T) {
	// TODO: Combine with above... test best practice rules.
	basepath := testutil.RelPath("test/data/validator")
	searchpath := testutil.RelPath("test/data/validator/best-practices")
	files, err := ioutil.ReadDir(searchpath)
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
				fn := func(a string, b bool) string { return a }
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
			opts.BestPractices = true
			opts.ErrorHandler = &handler
			v, _ := NewValidator(reader, opts)
			result, _ := v.Validate()
			_ = result
			if handler.expectErrorCount == 0 {
				t.Errorf("feed did not contain any test cases")
			}
		})
	}
}
