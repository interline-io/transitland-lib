package rt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	mapset "github.com/deckarep/golang-set/v2"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/interline-io/transitland-lib/tlcsv"
)

// NewValidatorFromReader returns a Validator with data from a Reader.
func NewValidatorFromReader(reader adapters.Reader) (*Validator, error) {
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
	r, err := tlcsv.NewReader(testpath.RelPath("testdata/rt/bart-rt.zip"))
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
	msg, err := ReadFile(testpath.RelPath("testdata/rt/bart-trip-updates.pb"))
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
	msg, err := ReadFile(testpath.RelPath("testdata/rt/bart-trip-updates.pb"))
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

func TestValidatorErrors(t *testing.T) {
	rp := func(p string) string {
		return testpath.RelPath(filepath.Join("testdata/rt/", p))
	}
	rpe := func(p string) string {
		return testpath.RelPath(filepath.Join("testdata/rt/errors", p))
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
