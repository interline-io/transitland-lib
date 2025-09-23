package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestStop_ConditionalErrors(t *testing.T) {
	tests := []struct {
		name     string
		stop     Stop
		wantErrs bool
	}{
		{
			name: "valid stop with stop_access=0",
			stop: Stop{
				StopID:        tt.NewString("stop1"),
				LocationType:  tt.NewInt(0),
				ParentStation: tt.NewKey("station1"),
				StopAccess:    tt.NewInt(0),
			},
			wantErrs: false,
		},
		{
			name: "valid stop with stop_access=1",
			stop: Stop{
				StopID:        tt.NewString("stop2"),
				LocationType:  tt.NewInt(0),
				ParentStation: tt.NewKey("station1"),
				StopAccess:    tt.NewInt(1),
			},
			wantErrs: false,
		},
		{
			name: "invalid stop_access on station",
			stop: Stop{
				StopID:       tt.NewString("station1"),
				LocationType: tt.NewInt(1),
				StopAccess:   tt.NewInt(1),
			},
			wantErrs: true,
		},
		{
			name: "invalid stop_access on entrance",
			stop: Stop{
				StopID:        tt.NewString("entrance1"),
				LocationType:  tt.NewInt(2),
				ParentStation: tt.NewKey("station1"),
				StopAccess:    tt.NewInt(1),
			},
			wantErrs: true,
		},
		{
			name: "invalid stop_access on generic node",
			stop: Stop{
				StopID:        tt.NewString("node1"),
				LocationType:  tt.NewInt(3),
				ParentStation: tt.NewKey("station1"),
				StopAccess:    tt.NewInt(1),
			},
			wantErrs: true,
		},
		{
			name: "invalid stop_access on boarding area",
			stop: Stop{
				StopID:        tt.NewString("area1"),
				LocationType:  tt.NewInt(4),
				ParentStation: tt.NewKey("station1"),
				StopAccess:    tt.NewInt(1),
			},
			wantErrs: true,
		},
		{
			name: "invalid stop_access without parent_station",
			stop: Stop{
				StopID:       tt.NewString("stop3"),
				LocationType: tt.NewInt(0),
				StopAccess:   tt.NewInt(1),
			},
			wantErrs: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tc.stop.ConditionalErrors()
			if (len(errs) > 0) != tc.wantErrs {
				t.Errorf("Stop.ConditionalErrors() got %v errors, want errors: %v", len(errs), tc.wantErrs)
			}
		})
	}
}
