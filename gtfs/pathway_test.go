package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestPathway_ConditionalErrors(t *testing.T) {
	tests := []struct {
		name     string
		pathway  Pathway
		wantErrs bool
	}{
		{
			name: "valid pathway",
			pathway: Pathway{
				PathwayID:       tt.NewString("path1"),
				FromStopID:      tt.NewString("stop1"),
				ToStopID:        tt.NewString("stop2"),
				PathwayMode:     tt.NewInt(1),
				IsBidirectional: tt.NewInt(1),
			},
			wantErrs: false,
		},
		{
			name: "invalid bidirectional exit gate",
			pathway: Pathway{
				PathwayID:       tt.NewString("path4"),
				FromStopID:      tt.NewString("stop7"),
				ToStopID:        tt.NewString("stop8"),
				PathwayMode:     tt.NewInt(7),
				IsBidirectional: tt.NewInt(1),
			},
			wantErrs: true,
		},
		{
			name: "valid max_slope for walkway",
			pathway: Pathway{
				PathwayID:       tt.NewString("path5"),
				FromStopID:      tt.NewString("stop9"),
				ToStopID:        tt.NewString("stop10"),
				PathwayMode:     tt.NewInt(1),
				IsBidirectional: tt.NewInt(1),
				MaxSlope:        tt.NewFloat(0.05),
			},
			wantErrs: false,
		},
		{
			name: "invalid max_slope for elevator",
			pathway: Pathway{
				PathwayID:       tt.NewString("path6"),
				FromStopID:      tt.NewString("stop11"),
				ToStopID:        tt.NewString("stop12"),
				PathwayMode:     tt.NewInt(5),
				IsBidirectional: tt.NewInt(1),
				MaxSlope:        tt.NewFloat(0.05),
			},
			wantErrs: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tc.pathway.ConditionalErrors()
			if (len(errs) > 0) != tc.wantErrs {
				t.Errorf("Pathway.ConditionalErrors() got %v errors, want errors: %v", len(errs), tc.wantErrs)
			}
		})
	}
}
