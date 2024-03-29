package rt

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/stretchr/testify/assert"

	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlcsv"
)

// NewValidatorFromReader returns a Validator with data from a Reader.
func NewValidatorFromReader(reader tl.Reader) (*Validator, error) {
	fi := NewValidator()
	cp, err := copier.NewCopier(reader, &empty.Writer{}, copier.Options{})
	if err != nil {
		return nil, err
	}
	if err := cp.AddExtension(fi); err != nil {
		return nil, err
	}
	cpResult := cp.Copy()
	if cpResult.WriteError != nil {
		return nil, cpResult.WriteError
	}
	return fi, nil
}

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

	t.Run("midday", func(t *testing.T) {
		// Tuesday, Nov 7 2023 17:30:00
		tz, _ := time.LoadLocation("America/Los_Angeles")
		now := time.Date(2023, 11, 7, 17, 30, 0, 0, tz)
		stats, err := ex.TripUpdateStats(now, msg)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, 4, len(stats))
		for _, stat := range stats {
			if stat.RouteID == "L1" {
				assert.ElementsMatch(t, []string{"127", "126", "125"}, stat.TripScheduledIDs)
				assert.Equal(t, 3, stat.TripScheduledCount)
				assert.Equal(t, 3, stat.TripMatchCount)
			} else if stat.RouteID == "L4" {
				assert.ElementsMatch(t, []string{"411", "410", "412"}, stat.TripScheduledIDs)
				assert.Equal(t, 3, stat.TripScheduledCount)
				assert.Equal(t, 3, stat.TripMatchCount)
			} else if stat.RouteID == "L3" {
				assert.ElementsMatch(t, []string{"308", "311", "309", "312", "310"}, stat.TripScheduledIDs)
				assert.Equal(t, 5, stat.TripScheduledCount)
				assert.Equal(t, 4, stat.TripMatchCount)
			} else if stat.RouteID == "B7" {
				assert.ElementsMatch(t, []string{"710", "709"}, stat.TripScheduledIDs)
				assert.Equal(t, 2, stat.TripScheduledCount)
				assert.Equal(t, 2, stat.TripMatchCount)
			} else {
				t.Errorf("route %s not scheduled", stat.RouteID)
			}
		}
	})
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
	sor := func(a, b string) string {
		if a != "" {
			return a
		}
		return b
	}
	// ms := func(vals ...int) mapset.Set[int] {
	// 	a := mapset.NewSet[int]()
	// 	for _, v := range vals {
	// 		a.Add(v)
	// 	}
	// 	return a
	// }

	type match struct {
		field string
		err   string
		warn  string
	}
	type testCase struct {
		name    string
		matches []match
		static  string
		rt      string
	}
	tcs := []testCase{}

	// Automatic
	fns, err := os.ReadDir(rpe(""))
	if err != nil {
		t.Fatal(err)
	}
	for _, fn := range fns {
		tcs = append(tcs, testCase{rt: rpe(fn.Name())})
	}
	for _, tc := range tcs {
		t.Run(sor(tc.name, tc.rt), func(t *testing.T) {
			var expMatches []match
			if fnSplit := strings.Split(filepath.Base(tc.rt), "."); len(fnSplit) > 2 {
				fnCode := fnSplit[0]
				expMatches = append(expMatches, match{
					err:   fnCode,
					field: strings.ReplaceAll(fnSplit[1], "-", "."),
				})
			}
			expMatches = append(expMatches, tc.matches...)

			// Read RT
			msg, err := ReadFile(tc.rt)
			if err != nil {
				t.Fatal(err)
			}
			// Read static
			r, err := tlcsv.NewReader(sor(tc.static, rp("ct.zip")))
			if err != nil {
				t.Fatal(err)
			}
			// Validate
			cp, err := copier.NewCopier(r, &empty.Writer{}, copier.Options{})
			if err != nil {
				t.Fatal(err)
			}
			ex := NewValidator()
			cp.AddExtension(ex)
			result := cp.Copy()
			_ = result
			// Validate feed message
			rterrs := ex.ValidateFeedMessage(msg, nil)

			// Check results
			foundSet := mapset.NewSet[match]()
			for _, rterr := range rterrs {
				if a, ok := rterr.(*RealtimeError); ok {
					foundSet.Add(match{
						field: a.bc.Field,
						err:   a.bc.ErrorCode,
					})
				}
				if a, ok := rterr.(*RealtimeWarning); ok {
					foundSet.Add(match{
						field: a.bc.Field,
						warn:  a.bc.ErrorCode,
					})
				}
			}
			expSet := mapset.NewSet[match]()
			for _, m := range expMatches {
				expSet.Add(m)
			}
			for _, exp := range expMatches {
				if !foundSet.Contains(exp) {
					t.Errorf("expected to find error %v", exp)
				}
			}
			for got := range foundSet.Iter() {
				if !expSet.Contains(got) {
					t.Errorf("got unexpected error %v", got)
				}
			}
		})
	}
}
