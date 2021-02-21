package validator

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlcsv"
)

//////////// helpers /////////////

// getExpectErrors gets any ExpectError specified by an Entity.
func getExpectErrors(ent tl.Entity) []testutil.ExpectError {
	ret := []testutil.ExpectError{}
	ex := ent.Extra()
	value, ok := ex["expect_error"]
	if len(value) == 0 || !ok {
		return ret
	}
	for _, v := range strings.Split(value, "|") {
		ee := testutil.ParseExpectError(v)
		if ee.Filename == "" {
			ee.Filename = ent.Filename()
		}
		if ee.EntityID == "" {
			ee.EntityID = ent.EntityID()
		}
		ret = append(ret, ee)
	}
	return ret
}

func checkErrors(expecterrs []testutil.ExpectError, errs []error, t *testing.T) {
	s1 := []string{}
	for _, err := range errs {
		s1 = append(s1, fmt.Sprintf("%#v", err))
	}
	if len(errs) > len(expecterrs) {
		s2 := []string{}
		for _, err := range expecterrs {
			s2 = append(s2, fmt.Sprintf("%#v", err))
		}

		t.Errorf("got %d errors/warnings, more than the expected expected %d, got: %s expect: %s", len(errs), len(expecterrs), strings.Join(s1, " "), strings.Join(s2, " "))
		return
	}
	for _, expect := range expecterrs {
		expect.Filename = ""
		expect.EntityID = ""
		if !expect.Match(errs) {
			t.Errorf("did not find match for expected error %#v, got: %s", expect, strings.Join(s1, " "))
		}
	}
}

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
	checkErrors(expecterrs, errs, cr.t)
}

func (cr *testErrorHandler) HandleEntityErrors(ent tl.Entity, errs []error, warns []error) {
	errs = append(errs, warns...)
	expecterrs := getExpectErrors(ent)
	cr.expectErrorCount += len(expecterrs)
	checkErrors(expecterrs, errs, cr.t)
}

//////////////

func TestEntityErrors(t *testing.T) {
	reader, err := tlcsv.NewReader("../test/data/bad-entities")
	if err != nil {
		t.Error(err)
	}
	if err := reader.Open(); err != nil {
		t.Error(err)
	}
	testutil.AllEntities(reader, func(ent tl.Entity) {
		t.Run(fmt.Sprintf("%s:%s", ent.Filename(), ent.EntityID()), func(t *testing.T) {
			errs := ent.Errors()
			errs = append(errs, ent.Warnings()...)
			expecterrs := getExpectErrors(ent)
			checkErrors(expecterrs, errs, t)
		})
	})
	if err := reader.Close(); err != nil {
		t.Error(err)
	}
}

func TestValidator_Validate(t *testing.T) {
	basepath := "../test/data/validator-examples"
	searchpath := "../test/data/validator-examples/errors"
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
