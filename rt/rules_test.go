package rt

import (
	"fmt"
	"testing"
)

func newfi() *FeedInfo {
	r := newReader()
	fi, err := NewFeedInfoFromReader(r)
	if err != nil {
		panic(err)
	}
	return fi
}

func TestValidateHeader(t *testing.T) {
	fi := newfi()
	msg, err := readmsg("../testdata/rt/example.pb")
	if err != nil {
		t.Error(err)
	}
	header := msg.GetHeader()
	errs := ValidateHeader(fi, header, &msg)
	for _, err := range errs {
		fmt.Println(err)
	}
}

func TestValidateTripUpdate(t *testing.T) {
	fi := newfi()
	msg, err := readmsg("../testdata/rt/example.pb")
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
	errs := ValidateTripUpdate(fi, trip, &msg)
	for _, err := range errs {
		fmt.Println(err)
	}
}

func TestValidateVehiclePosition(t *testing.T) {

}

func TestValidateAlert(t *testing.T) {

}
