package model

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/gbfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

// GBFS defines parking_hoop as a boolean; the schema has always exposed an Int.
func TestGbfsStationInformation_ParkingHoop(t *testing.T) {
	for _, tc := range []struct {
		name   string
		hoop   tt.Bool
		expect tt.Int
	}{
		{"present", tt.NewBool(true), tt.NewInt(1)},
		{"absent", tt.NewBool(false), tt.NewInt(0)},
		{"unset", tt.Bool{}, tt.Int{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := GbfsStationInformation{StationInformation: &gbfs.StationInformation{ParkingHoop: tc.hoop}}
			assert.Equal(t, tc.expect, s.ParkingHoop())
		})
	}
}

// A station's status carries its feed, which is what resolves the vehicle types
// its availability names.
func TestGbfsStationInformation_Status(t *testing.T) {
	feed := &GbfsFeed{GbfsFeed: &gbfs.GbfsFeed{
		StationStatus: []*gbfs.StationStatus{{
			StationID:             tt.NewString("s1"),
			VehicleDocksAvailable: []*gbfs.VehicleDockAvailable{{VehicleTypeIDs: tt.NewStrings([]string{"bike"})}},
		}},
		VehicleTypes: []*gbfs.VehicleType{{VehicleTypeID: tt.NewString("bike")}},
	}}
	s := GbfsStationInformation{Feed: feed, StationInformation: &gbfs.StationInformation{StationID: tt.NewString("s1")}}
	docks := s.Status().VehicleDocksAvailable()
	if assert.Len(t, docks, 1) {
		assert.Len(t, docks[0].VehicleTypes(), 1)
	}
}
