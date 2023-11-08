package rt

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

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
