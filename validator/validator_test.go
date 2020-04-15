package validator

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
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

type hasContext interface {
	Context() *causes.Context
}

func testFeed(t *testing.T, reader gotransit.Reader) {
	expecterrs := []testutil.ExpectError{}
	gex := func(ent gotransit.Entity) {
		expecterrs = append(expecterrs, testutil.GetExpectErrors(ent)...)
	}
	testutil.AllEntities(reader, gex)
	v, _ := NewValidator(reader)
	res := v.Validate()
	errs := res.Errors
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
	if len(expecterrs) != len(errs) {
		t.Errorf("test had uncaught errors")
		for _, expect := range expecterrs {
			fmt.Printf("expect err: %#v\n", expect)
		}
		for _, err := range errs {
			fmt.Printf("got err: %#v\n", err)
		}
	}
}

////////////

func TestValidator_EntityErrors(t *testing.T) {
	reader, err := gtcsv.NewReader("../testdata/bad-entities")
	if err != nil {
		t.Error(err)
	}
	if err := reader.Open(); err != nil {
		t.Error(err)
	}
	readAllEntities(t, reader, checkEntity)
	if err := reader.Close(); err != nil {
		t.Error(err)
	}
}

func checkEntity(ent gotransit.Entity, t *testing.T) {
	errs := ent.Errors()
	errs = append(errs, ent.Warnings()...)
	expecterrs := testutil.GetExpectErrors(ent)
	if len(errs) == 0 && len(expecterrs) == 0 {
		return
	}
	t.Run(fmt.Sprintf("%s:%s", ent.Filename(), ent.EntityID()), func(t2 *testing.T) {
		if len(expecterrs) != len(errs) {
			t2.Error("got errors that were not listed in expect_errors")
			for _, expect := range expecterrs {
				fmt.Printf("expect err: %#v\n", expect)
			}
			for _, err := range errs {
				fmt.Printf("got err: %#v\n", err)
			}
		}
		for _, expect := range expecterrs {
			expect.Filename = ""
			expect.EntityID = ""
			if !expect.Match(errs) {
				t2.Error("did not find:", expect, "got:", errs)
			}
		}
	})
}

// readAllEntities checks that all expected Entity errors are present.
func readAllEntities(t *testing.T, r gotransit.Reader, cb func(gotransit.Entity, *testing.T)) {
	for ent := range r.Agencies() {
		cb(&ent, t)
	}
	for ent := range r.Stops() {
		cb(&ent, t)
	}
	for ent := range r.Routes() {
		cb(&ent, t)
	}
	for ent := range r.Trips() {
		cb(&ent, t)
	}
	for ent := range r.StopTimes() {
		cb(&ent, t)
	}
	for ent := range r.Calendars() {
		cb(&ent, t)
	}
	for ent := range r.CalendarDates() {
		cb(&ent, t)
	}
	for ent := range r.FareAttributes() {
		cb(&ent, t)
	}
	for ent := range r.FareRules() {
		cb(&ent, t)
	}
	for ent := range r.FeedInfos() {
		cb(&ent, t)
	}
	for ent := range r.Shapes() {
		cb(&ent, t)
	}
	for ent := range r.Transfers() {
		cb(&ent, t)
	}
	for ent := range r.Frequencies() {
		cb(&ent, t)
	}
}
