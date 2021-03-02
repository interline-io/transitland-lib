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

func exampleReader(basepath string, overlaypath string) *tlcsv.Reader {
	reader, err := tlcsv.NewReader(".")
	if err != nil {
		return nil
	}
	reader.Adapter = tlcsv.NewOverlayAdapter(overlaypath, basepath)
	return reader
}

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

func (cr *testErrorHandler) HandleEntityErrors(ent tl.Entity, errs []error, warns []error) {
	errs = append(errs, warns...)
	expecterrs := testutil.GetExpectErrors(ent)
	cr.expectErrorCount += len(expecterrs)
	testutil.CheckErrors(expecterrs, errs, cr.t)
}

//////////////

func TestValidator_Validate(t *testing.T) {
	basepath := testutil.RelPath("test/data/validator-examples")
	searchpath := testutil.RelPath("test/data/validator-examples/errors")
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
			v, _ := NewValidator(reader)
			v.Copier.ErrorHandler = &handler
			errs, warns := v.Validate()
			_ = errs
			_ = warns
			if handler.expectErrorCount == 0 {
				t.Errorf("feed did not contain any test cases")
			}
		})
	}
}
