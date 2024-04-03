package rt

import (
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

	// Tuesday, Nov 7 2023 17:30:00
	tz, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Date(2023, 11, 7, 17, 30, 0, 0, tz)
	stats, err := ex.TripUpdateStats(now, msg)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 4, len(stats))
	byRoute := map[statAggKey][]TripUpdateStats{}
	for _, stat := range stats {
		k := statAggKey{RouteID: stat.RouteID, AgencyID: stat.AgencyID}
		byRoute[k] = append(byRoute[k], stat)
	}
	expectStats := map[statAggKey]TripUpdateStats{
		{AgencyID: "CT", RouteID: "L1"}: {
			TripScheduledIDs:        []string{"127", "126", "125"},
			TripScheduledCount:      3,
			TripScheduledMatched:    3,
			TripScheduledNotMatched: 0,
			TripRtIDs:               []string{"124", "125", "126", "127", "128", "129"},
			TripRtCount:             6,
			TripRtMatched:           3,
			TripRtNotMatched:        3,
		},
		{AgencyID: "CT", RouteID: "L4"}: {
			TripScheduledIDs:        []string{"411", "410", "412"},
			TripScheduledCount:      3,
			TripScheduledMatched:    3,
			TripScheduledNotMatched: 0,
			TripRtIDs:               []string{"410", "411", "412", "413", "414"},
			TripRtCount:             5,
			TripRtMatched:           3,
			TripRtNotMatched:        2,
		},
		{AgencyID: "CT", RouteID: "L3"}: {
			TripScheduledIDs:        []string{"308", "311", "309", "312", "310"},
			TripScheduledCount:      5,
			TripScheduledMatched:    4,
			TripScheduledNotMatched: 1,
			TripRtIDs:               []string{"310", "311", "312", "308"},
			TripRtCount:             4,
			TripRtMatched:           4,
			TripRtNotMatched:        0,
		},
		{AgencyID: "CT", RouteID: "B7"}: {
			TripScheduledIDs:        []string{"710", "709"},
			TripScheduledCount:      2,
			TripScheduledMatched:    2,
			TripScheduledNotMatched: 0,
			TripRtIDs:               []string{"709", "710", "711", "712"},
			TripRtCount:             4,
			TripRtMatched:           2,
			TripRtNotMatched:        2,
		},
	}
	for k, expect := range expectStats {
		t.Run(fmt.Sprintf("%s:%s", k.AgencyID, k.RouteID), func(t *testing.T) {
			rstats := byRoute[k]
			if len(rstats) != 1 {
				t.Fatal("expected 1 stat")
			}
			stat := rstats[0]
			assert.ElementsMatch(t, expect.TripScheduledIDs, stat.TripScheduledIDs, "TripScheduledIDs")
			assert.Equal(t, expect.TripScheduledCount, stat.TripScheduledCount, "TripScheduledCount")
			assert.Equal(t, expect.TripScheduledMatched, stat.TripScheduledMatched, "TripScheduledMatched")
			assert.Equal(t, expect.TripScheduledNotMatched, stat.TripScheduledNotMatched, "TripScheduledNotMatched")
			assert.ElementsMatch(t, expect.TripRtIDs, stat.TripRtIDs, "TripRtIDs")
			assert.Equal(t, expect.TripRtCount, stat.TripRtCount, "TripRtCount")
			assert.Equal(t, expect.TripRtMatched, stat.TripRtMatched, "TripRtMatched")
			assert.Equal(t, expect.TripRtNotMatched, stat.TripRtNotMatched, "TripRtNotMatched")
		})
	}
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

	// Tuesday, Nov 7 2023 17:30:00
	tz, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Date(2023, 11, 7, 17, 30, 0, 0, tz)
	stats, err := ex.VehiclePositionStats(now, msg)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 4, len(stats))
	byRoute := map[statAggKey][]VehiclePositionStats{}
	for _, stat := range stats {
		k := statAggKey{RouteID: stat.RouteID, AgencyID: stat.AgencyID}
		byRoute[k] = append(byRoute[k], stat)
	}
	expectStats := map[statAggKey]VehiclePositionStats{
		{AgencyID: "CT", RouteID: "L1"}: {
			TripScheduledIDs:        []string{"125", "126", "127"},
			TripScheduledCount:      3,
			TripScheduledMatched:    3,
			TripScheduledNotMatched: 0,
			TripRtIDs:               []string{"124", "125", "126", "127"},
			TripRtCount:             4,
			TripRtMatched:           3,
			TripRtNotMatched:        1,
		},
		{AgencyID: "CT", RouteID: "L4"}: {
			TripScheduledIDs:        []string{"411", "410", "412"},
			TripScheduledCount:      3,
			TripScheduledMatched:    3,
			TripScheduledNotMatched: 0,
			TripRtIDs:               []string{"410", "411", "412", "414"},
			TripRtCount:             4,
			TripRtMatched:           3,
			TripRtNotMatched:        1,
		},
		{AgencyID: "CT", RouteID: "L3"}: {
			TripScheduledIDs:        []string{"308", "311", "309", "312", "310"},
			TripScheduledCount:      5,
			TripScheduledMatched:    4,
			TripScheduledNotMatched: 1,
			TripRtIDs:               []string{"310", "311", "312", "308"},
			TripRtCount:             4,
			TripRtMatched:           4,
			TripRtNotMatched:        0,
		},
		{AgencyID: "CT", RouteID: "B7"}: {
			TripScheduledIDs:        []string{"710", "709"},
			TripScheduledCount:      2,
			TripScheduledMatched:    2,
			TripScheduledNotMatched: 0,
			TripRtIDs:               []string{"709", "710"},
			TripRtCount:             2,
			TripRtMatched:           2,
			TripRtNotMatched:        0,
		},
	}
	for k, expect := range expectStats {
		t.Run(fmt.Sprintf("%s:%s", k.AgencyID, k.RouteID), func(t *testing.T) {
			rstats := byRoute[k]
			if len(rstats) != 1 {
				t.Fatal("expected 1 stat")
			}
			stat := rstats[0]
			assert.ElementsMatch(t, expect.TripScheduledIDs, stat.TripScheduledIDs, "TripScheduledIDs")
			assert.Equal(t, expect.TripScheduledCount, stat.TripScheduledCount, "TripScheduledCount")
			assert.Equal(t, expect.TripScheduledMatched, stat.TripScheduledMatched, "TripScheduledMatched")
			assert.Equal(t, expect.TripScheduledNotMatched, stat.TripScheduledNotMatched, "TripScheduledNotMatched")
			assert.ElementsMatch(t, expect.TripRtIDs, stat.TripRtIDs, "TripRtIDs")
			assert.Equal(t, expect.TripRtCount, stat.TripRtCount, "TripRtCount")
			assert.Equal(t, expect.TripRtMatched, stat.TripRtMatched, "TripRtMatched")
			assert.Equal(t, expect.TripRtNotMatched, stat.TripRtNotMatched, "TripRtNotMatched")
		})
	}
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
