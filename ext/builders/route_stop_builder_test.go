package builders

import (
	"testing"

	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/stretchr/testify/assert"
)

func newMockCopier(url string, exts ...any) (*copier.Copier, *direct.Writer, error) {
	reader, err := tlcsv.NewReader(url)
	if err != nil {
		return nil, nil, err
	}
	writer := direct.NewWriter()
	cpOpts := copier.Options{}
	for _, e := range exts {
		cpOpts.AddExtension(e)
	}
	cp, err := copier.NewCopier(reader, writer, cpOpts)
	if err != nil {
		return nil, nil, err
	}
	return cp, writer, nil
}

func TestRouteStopBuilder(t *testing.T) {
	e := NewRouteStopBuilder()
	cp, writer, err := newMockCopier(testutil.ExampleFeedBART.URL, e)
	if err != nil {
		t.Fatal(err)
	}
	cpr := cp.Copy()
	if cpr.WriteError != nil {
		t.Fatal(err)
	}
	routeStops := []*RouteStop{}
	for _, ent := range writer.Reader.OtherList {
		if v, ok := ent.(*RouteStop); ok {
			routeStops = append(routeStops, v)
		}
	}
	testcases := []struct {
		Name     string
		AgencyID string
		RouteID  string
		StopIDs  []string
	}{
		{"BART-01", "BART", "01", []string{"SFIA", "PITT", "WOAK", "EMBR", "CIVC", "COLM", "19TH_N", "ROCK", "MONT", "DALY", "NCON", "12TH", "POWL", "SBRN", "PCTR", "24TH", "GLEN", "CONC", "WCRK", "MCAR_S", "BALB", "LAFY", "ORIN", "16TH", "MLBR", "PHIL", "19TH", "SSAN", "ANTC", "MCAR"}},
		{"BART-07", "BART", "07", []string{"DBRK", "MONT", "CIVC", "BALB", "DALY", "COLM", "SSAN", "DELN", "19TH", "POWL", "16TH", "GLEN", "SBRN", "MLBR", "MCAR", "RICH", "WOAK", "ASHB", "MCAR_S", "12TH", "EMBR", "24TH", "19TH_N", "PLZA", "NBRK"}},
		{"BART-19", "BART", "19", []string{"COLS", "OAKL"}},
	}
	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			foundStopIDs := []string{}
			for _, ent := range routeStops {
				if ent.AgencyID == tc.AgencyID && ent.RouteID == tc.RouteID {
					foundStopIDs = append(foundStopIDs, ent.StopID)
				}
			}
			assert.ElementsMatch(t, tc.StopIDs, foundStopIDs)
		})
	}
}
