package validator

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/gtcsv"
	_ "github.com/interline-io/gotransit/gtcsv"
	"github.com/interline-io/gotransit/internal/testutil"
)

func exampleReader(basepath string, overlaypath string) *gtcsv.Reader {
	reader, err := gtcsv.NewReader(".")
	if err != nil {
		return nil
	}
	reader.Adapter = gtcsv.NewOverlayAdapter(overlaypath, basepath)
	return reader
}

func TestValidator_Validate(t *testing.T) {
	basepath := "../testdata/validator-examples"
	searchpath := "../testdata/validator-examples/errors"
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
			testFeed(t, reader)
		})
	}
}

func testFeed(t *testing.T, reader gotransit.Reader) {
	expecterrs := testutil.CollectExpectErrors(reader)
	v, _ := NewValidator(reader)
	errs, warns := v.Validate()
	_ = warns
	if len(expecterrs) == 0 {
		t.Errorf("test case does not contain any test cases or warnings")
	}
	for _, expect := range expecterrs {
		t.Run(fmt.Sprintf("%s:%s", expect.ErrorType, expect.Field), func(t *testing.T) {
			if !expect.Match(errs) {
				got := []string{}
				for _, i := range errs {
					got = append(got, i.Error())
				}
				t.Errorf("expected error %s not found, got: %s", expect.String(), strings.Join(got, ", "))
			}
		})
	}
}
