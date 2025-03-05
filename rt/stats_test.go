package rt

import (
	"fmt"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/stretchr/testify/assert"
)

func TestTripUpdateStats(t *testing.T) {
	r, err := tlcsv.NewReader(testpath.RelPath("testdata/rt/ct.zip"))
	if err != nil {
		t.Fatal(err)
	}
	msg, err := ReadFile(testpath.RelPath("testdata/rt/ct-trip-stats.json"))
	if err != nil {
		t.Error(err)
	}
	ex := NewValidator()
	cpOpts := copier.Options{}
	cpOpts.AddExtension(ex)
	cp, err := copier.NewCopier(r, &empty.Writer{}, copier.Options{})
	if err != nil {
		t.Fatal(err)
	}
	result := cp.Copy()
	_ = result

	// Tuesday, Nov 7 2023 17:30:00
	tz, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Date(2023, 11, 7, 17, 30, 0, 0, tz)
	stats, err := ex.TripUpdateStats(now, msg)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 10, len(stats), "stat count")
	byRoute := map[statAggKey][]RTTripStat{}
	for _, stat := range stats {
		k := statAggKey{RouteID: stat.RouteID, AgencyID: stat.AgencyID}
		byRoute[k] = append(byRoute[k], stat)
	}
	expectStats := map[statAggKey]RTTripStat{
		{AgencyID: "", RouteID: ""}: {
			TripRtNotFoundIDs:   []string{"notfound-noroute"},
			TripRtNotFoundCount: 1,
		},
		{AgencyID: "CT", RouteID: "L1"}: {
			TripScheduledIDs:        []string{"127", "126", "125"},
			TripScheduledCount:      3,
			TripScheduledMatched:    3,
			TripScheduledNotMatched: 0,
			TripRtIDs:               []string{"124", "125", "126", "127", "128", "129"},
			TripRtCount:             6,
			TripRtMatched:           3,
			TripRtNotMatched:        3,
			TripRtAddedIDs:          []string{"124added"},
			TripRtAddedCount:        1,
			TripRtNotFoundIDs:       []string{"notfound"},
			TripRtNotFoundCount:     1,
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
			assert.ElementsMatch(t, expect.TripRtAddedIDs, stat.TripRtAddedIDs, "TripRtIDs")
			assert.ElementsMatch(t, expect.TripRtNotFoundIDs, stat.TripRtNotFoundIDs, "TripRtNotFoundIDs")
			assert.Equal(t, expect.TripRtCount, stat.TripRtCount, "TripRtCount")
			assert.Equal(t, expect.TripRtMatched, stat.TripRtMatched, "TripRtMatched")
			assert.Equal(t, expect.TripRtNotMatched, stat.TripRtNotMatched, "TripRtNotMatched")
			assert.Equal(t, expect.TripRtAddedCount, stat.TripRtAddedCount, "TripRtAddedCount")
			assert.Equal(t, expect.TripRtNotFoundCount, stat.TripRtNotFoundCount, "TripRtNotFoundCount")
		})
	}
}

func TestVehiclePositionStats(t *testing.T) {
	r, err := tlcsv.NewReader(testpath.RelPath("testdata/rt/ct.zip"))
	if err != nil {
		t.Fatal(err)
	}
	msg, err := ReadFile(testpath.RelPath("testdata/rt/ct-vehicle-stats.json"))
	if err != nil {
		t.Error(err)
	}
	ex := NewValidator()
	cpOpts := copier.Options{}
	cpOpts.AddExtension(ex)
	cp, err := copier.NewCopier(r, &empty.Writer{}, cpOpts)
	if err != nil {
		t.Fatal(err)
	}
	result := cp.Copy()
	_ = result

	// Tuesday, Nov 7 2023 17:30:00
	tz, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Date(2023, 11, 7, 17, 30, 0, 0, tz)
	stats, err := ex.VehiclePositionStats(now, msg)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 10, len(stats))
	byRoute := map[statAggKey][]RTTripStat{}
	for _, stat := range stats {
		k := statAggKey{RouteID: stat.RouteID, AgencyID: stat.AgencyID}
		byRoute[k] = append(byRoute[k], stat)
	}
	expectStats := map[statAggKey]RTTripStat{
		{AgencyID: "", RouteID: ""}: {
			TripRtNotFoundIDs:   []string{"notfound-noroute"},
			TripRtNotFoundCount: 1,
		},
		{AgencyID: "CT", RouteID: "L1"}: {
			TripScheduledIDs:        []string{"125", "126", "127"},
			TripScheduledCount:      3,
			TripScheduledMatched:    3,
			TripScheduledNotMatched: 0,
			TripRtIDs:               []string{"124", "125", "126", "127"},
			TripRtCount:             4,
			TripRtMatched:           3,
			TripRtNotMatched:        1,
			TripRtAddedIDs:          []string{"124added"},
			TripRtAddedCount:        1,
			TripRtNotFoundIDs:       []string{"notfound"},
			TripRtNotFoundCount:     1,
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
			assert.ElementsMatch(t, expect.TripRtAddedIDs, stat.TripRtAddedIDs, "TripRtIDs")
			assert.ElementsMatch(t, expect.TripRtNotFoundIDs, stat.TripRtNotFoundIDs, "TripRtNotFoundIDs")
			assert.Equal(t, expect.TripRtCount, stat.TripRtCount, "TripRtCount")
			assert.Equal(t, expect.TripRtMatched, stat.TripRtMatched, "TripRtMatched")
			assert.Equal(t, expect.TripRtNotMatched, stat.TripRtNotMatched, "TripRtNotMatched")
			assert.Equal(t, expect.TripRtAddedCount, stat.TripRtAddedCount, "TripRtAddedCount")
			assert.Equal(t, expect.TripRtNotFoundCount, stat.TripRtNotFoundCount, "TripRtNotFoundCount")
		})
	}
}
