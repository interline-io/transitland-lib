package rt

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set/v2"

	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tlcsv"
)

func newTestValidator() (*Validator, error) {
	r, err := tlcsv.NewReader(testutil.RelPath("test/data/rt/bart-rt.zip"))
	if err != nil {
		return nil, err
	}
	fi, err := NewValidatorFromReader(r)
	if err != nil {
		return nil, err
	}
	return fi, nil
}

func TestValidateHeader(t *testing.T) {
	fi, err := newTestValidator()
	if err != nil {
		t.Fatal(err)
	}
	msg, err := ReadFile(testutil.RelPath("test/data/rt/bart-trip-updates.pb"))
	if err != nil {
		t.Error(err)
	}
	header := msg.GetHeader()
	errs := fi.ValidateHeader(header, msg)
	for _, err := range errs {
		_ = err
	}
}

func TestValidateTripUpdate(t *testing.T) {
	fi, err := newTestValidator()
	if err != nil {
		t.Fatal(err)
	}
	msg, err := ReadFile(testutil.RelPath("test/data/rt/bart-trip-updates.pb"))
	if err != nil {
		t.Error(err)
	}
	ents := msg.GetEntity()
	if len(ents) == 0 {
		t.Error("no entities")
	}
	trip := ents[0].TripUpdate
	if trip == nil {
		t.Error("expected TripUpdate")
	}
	errs := fi.ValidateTripUpdate(trip, msg)
	for _, err := range errs {
		_ = err
	}
}

func TestValidateAlert(t *testing.T) {

}

func TestTripUpdateStats(t *testing.T) {
	r, err := tlcsv.NewReader(testutil.RelPath("test/data/rt/ct.zip"))
	if err != nil {
		t.Fatal(err)
	}
	msg, err := ReadFile(testutil.RelPath("test/data/rt/ct-trip-updates.pb"))
	if err != nil {
		t.Error(err)
	}
	cp, err := copier.NewCopier(r, &empty.Writer{}, copier.Options{})
	if err != nil {
		t.Fatal(err)
	}
	ex := NewValidator()
	cp.AddExtension(ex)
	result := cp.Copy()
	_ = result
	tz, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Date(2023, 11, 7, 5, 30, 0, 0, tz)
	stats, err := ex.TripUpdateStats(now, msg)
	if err != nil {
		t.Fatal(err)
	}
	jj, _ := json.Marshal(stats)
	fmt.Println(string(jj))
}

func TestVehiclePositionStats(t *testing.T) {
	r, err := tlcsv.NewReader(testutil.RelPath("test/data/rt/ct.zip"))
	if err != nil {
		t.Fatal(err)
	}
	msg, err := ReadFile(testutil.RelPath("test/data/rt/ct-vehicle-positions.pb"))
	if err != nil {
		t.Error(err)
	}
	cp, err := copier.NewCopier(r, &empty.Writer{}, copier.Options{})
	if err != nil {
		t.Fatal(err)
	}
	ex := NewValidator()
	cp.AddExtension(ex)
	result := cp.Copy()
	_ = result
	tz, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Date(2023, 11, 7, 5, 30, 0, 0, tz)
	stats, err := ex.VehiclePositionStats(now, msg)
	if err != nil {
		t.Fatal(err)
	}
	jj, _ := json.Marshal(stats)
	fmt.Println(string(jj))
}

func TestValidatorErrors(t *testing.T) {
	rp := func(p string) string {
		return testutil.RelPath(filepath.Join("test/data/rt/", p))
	}
	rpe := func(p string) string {
		return testutil.RelPath(filepath.Join("test/data/rt/errors", p))
	}
	ms := func(vals ...int) mapset.Set[int] {
		a := mapset.NewSet[int]()
		for _, v := range vals {
			a.Add(v)
		}
		return a
	}

	tcs := []struct {
		name          string
		field         string
		static        string
		rt            string
		expectError   mapset.Set[int]
		expectWarning mapset.Set[int]
	}{
		{
			name:        "ct-not-posix-1",
			static:      rp("ct.zip"),
			rt:          rpe("ct-not-posix-1.json"),
			expectError: ms(1),
		},
		{
			name:        "ct-not-posix-2",
			static:      rp("ct.zip"),
			rt:          rpe("ct-not-posix-2.json"),
			expectError: ms(1),
		},
		{
			name:        "ct-not-posix-3",
			static:      rp("ct.zip"),
			rt:          rpe("ct-not-posix-3.json"),
			expectError: ms(1),
		},
		{
			name:        "ct-not-posix-4",
			static:      rp("ct.zip"),
			rt:          rpe("ct-not-posix-4.json"),
			expectError: ms(1),
		},

		{
			name:        "ct-tu-invalid-stop",
			static:      rp("ct.zip"),
			rt:          rpe("ct-tu-invalid-stop.json"),
			expectError: ms(11),
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			msg, err := ReadRTJson(tc.rt)
			if err != nil {
				t.Fatal(err)
			}
			r, err := tlcsv.NewReader(tc.static)
			if err != nil {
				t.Fatal(err)
			}
			cp, err := copier.NewCopier(r, &empty.Writer{}, copier.Options{})
			if err != nil {
				t.Fatal(err)
			}
			ex := NewValidator()
			cp.AddExtension(ex)
			result := cp.Copy()
			_ = result
			rterrs := ex.ValidateFeedMessage(msg, nil)
			foundErrs := mapset.NewSet[int]()
			foundWarns := mapset.NewSet[int]()
			for _, rterr := range rterrs {
				if a, ok := rterr.(*RealtimeError); ok {
					foundErrs.Add(a.Code)
				}
				if a, ok := rterr.(*RealtimeWarning); ok {
					foundWarns.Add(a.Code)
				}
			}
			if tc.expectError != nil {
				if d := tc.expectError.SymmetricDifference(foundErrs); d.Cardinality() > 0 {
					t.Errorf("expected errors %v, got %v", tc.expectError, foundErrs)
				}
			}
			if tc.expectWarning != nil {
				if d := tc.expectWarning.SymmetricDifference(foundErrs); d.Cardinality() > 0 {
					t.Errorf("expected warnings %v, got %v", tc.expectWarning, foundWarns)
				}
			}
		})
	}
}
