package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tt"
)

func TestPathway_Errors(t *testing.T) {
	newPathway := func(fn func(*Pathway)) *Pathway {
		pathway := &Pathway{
			PathwayID:           tt.NewString("ok1"),
			FromStopID:          tt.NewString("stop1"),
			ToStopID:            tt.NewString("stop2"),
			PathwayMode:         tt.NewInt(1),
			IsBidirectional:     tt.NewInt(1),
			Length:              tt.NewFloat(10.5),
			TraversalTime:       tt.NewInt(60),
			StairCount:          tt.NewInt(5),
			MaxSlope:            tt.NewFloat(0.1),
			MinWidth:            tt.NewFloat(1.5),
			SignpostedAs:        tt.NewString("Sign 1"),
			ReverseSignpostedAs: tt.NewString("Reverse Sign 1"),
		}
		if fn != nil {
			fn(pathway)
		}
		return pathway
	}

	testcases := []struct {
		name           string
		entity         *Pathway
		expectedErrors []testutil.ExpectError
	}{
		{
			name:           "Valid pathway",
			entity:         newPathway(nil),
			expectedErrors: nil,
		},
		{
			name: "Invalid negative length",
			entity: newPathway(func(p *Pathway) {
				p.Length = tt.NewFloat(-2.0)
			}),
			expectedErrors: PE("InvalidFieldError:length"),
		},
		{
			name: "Invalid negative min_width",
			entity: newPathway(func(p *Pathway) {
				p.MinWidth = tt.NewFloat(-1.0)
			}),
			expectedErrors: PE("InvalidFieldError:min_width"),
		},
		{
			name: "Invalid zero min_width",
			entity: newPathway(func(p *Pathway) {
				p.MinWidth = tt.NewFloat(0.0)
			}),
			expectedErrors: PE("InvalidFieldError:min_width"),
		},

		{
			name: "Invalid max_slope for escalator (pathway_mode=2)",
			entity: newPathway(func(p *Pathway) {
				p.PathwayMode = tt.NewInt(2)
				p.MaxSlope = tt.NewFloat(0.05)
			}),
			expectedErrors: PE("InvalidFieldError:max_slope"),
		},
		{
			name: "Invalid max_slope for elevator (pathway_mode=4)",
			entity: newPathway(func(p *Pathway) {
				p.PathwayMode = tt.NewInt(4)
				p.MaxSlope = tt.NewFloat(0.05)
			}),
			expectedErrors: PE("InvalidFieldError:max_slope"),
		},
		{
			name: "Invalid pathway_mode zero",
			entity: newPathway(func(p *Pathway) {
				p.PathwayMode = tt.NewInt(0)
				p.MaxSlope = tt.Float{} // Clear max_slope to avoid conditional error
			}),
			expectedErrors: PE("InvalidFieldError:pathway_mode"),
		},
		{
			name: "Invalid pathway_mode eight",
			entity: newPathway(func(p *Pathway) {
				p.PathwayMode = tt.NewInt(8)
				p.MaxSlope = tt.Float{} // Clear max_slope to avoid conditional error
			}),
			expectedErrors: PE("InvalidFieldError:pathway_mode"),
		},
		{
			name: "Invalid pathway_mode negative",
			entity: newPathway(func(p *Pathway) {
				p.PathwayMode = tt.NewInt(-1)
				p.MaxSlope = tt.Float{} // Clear max_slope to avoid conditional error
			}),
			expectedErrors: PE("InvalidFieldError:pathway_mode"),
		},
		{
			name: "Invalid is_bidirectional value",
			entity: newPathway(func(p *Pathway) {
				p.IsBidirectional = tt.NewInt(2)
			}),
			expectedErrors: PE("InvalidFieldError:is_bidirectional"),
		},
		{
			name: "Exit gate (pathway_mode=7) cannot be bidirectional",
			entity: newPathway(func(p *Pathway) {
				p.PathwayMode = tt.NewInt(7)
				p.IsBidirectional = tt.NewInt(1)
				p.MaxSlope = tt.Float{} // Clear max_slope to avoid conditional error
			}),
			expectedErrors: PE("InvalidFieldError:is_bidirectional"),
		},
		{
			name: "Multiple invalid fields",
			entity: newPathway(func(p *Pathway) {
				p.Length = tt.NewFloat(-2.0)
				p.MinWidth = tt.NewFloat(-1.0)
			}),
			expectedErrors: PE("InvalidFieldError:length", "InvalidFieldError:min_width"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.entity)
			testutil.CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}
