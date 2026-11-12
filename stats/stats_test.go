package stats

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testreader"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadFeedVersionStats(t *testing.T) {
	// A version's stats read back as WriteFeedVersionStats wrote them.
	reader, err := tlcsv.NewReader(testreader.ExampleZip.URL)
	require.NoError(t, err)
	want, err := NewFeedStatsFromReader(reader)
	require.NoError(t, err)
	require.NotEmpty(t, want.ServiceLevels)
	require.NotEmpty(t, want.GeohashCells)
	require.True(t, want.Geometry.Valid, "the example feed has a hull")
	statsTestDBs(t, func(t *testing.T, atx tldb.Adapter) {
		ctx := context.Background()
		fv := testdb.CreateTestFeedVersion(atx, "")
		require.NoError(t, WriteFeedVersionStats(ctx, atx, want, fv.ID, WriteOptions{}))
		got, err := ReadFeedVersionStats(ctx, atx, fv.ID)
		require.NoError(t, err)
		assert.Equal(t, statsSummary(want), statsSummary(got))
	})
}

// statsSummary renders the stats a test compares, without the ids and
// timestamps a database assigns.
func statsSummary(s FeedVersionStats) map[string]any {
	w := s.ServiceWindow
	levels := []string{}
	for _, l := range s.ServiceLevels {
		levels = append(levels, fmt.Sprintf("%s %s %d %d %d %d %d %d %d", l.StartDate, l.EndDate, l.Monday, l.Tuesday, l.Wednesday, l.Thursday, l.Friday, l.Saturday, l.Sunday))
	}
	osids := map[string]string{}
	for _, v := range s.AgencyOnestopIDs {
		osids["agency "+v.EntityID] = v.OnestopID
	}
	for _, v := range s.RouteOnestopIDs {
		osids["route "+v.EntityID] = v.OnestopID
	}
	for _, v := range s.StopOnestopIDs {
		osids["stop "+v.EntityID] = v.OnestopID
	}
	files := map[string]string{}
	for _, v := range s.FileInfos {
		files[v.Name] = fmt.Sprintf("%d %d %s", v.Size, v.Rows, v.SHA1)
	}
	return map[string]any{
		"window":   fmt.Sprintf("%s %s %s %s %s %s", w.FeedStartDate, w.FeedEndDate, w.EarliestCalendarDate, w.LatestCalendarDate, w.FallbackWeek, w.DefaultTimezone),
		"levels":   levels,
		"osids":    osids,
		"files":    files,
		"geohash":  s.GeohashCells,
		"geometry": s.Geometry.FlatCoords(),
	}
}

// statsTestDBs runs fn against an in-memory SQLite database, then against the
// Postgres database at TL_TEST_DATABASE_URL, in a transaction it rolls back.
func statsTestDBs(t *testing.T, fn func(t *testing.T, atx tldb.Adapter)) {
	t.Run("sqlite", func(t *testing.T) {
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			fn(t, atx)
			return nil
		})
		require.NoError(t, err)
	})
	t.Run("postgres", func(t *testing.T) {
		dburl := os.Getenv("TL_TEST_DATABASE_URL")
		if dburl == "" {
			t.Skip("TL_TEST_DATABASE_URL is not set")
		}
		testdb.TempPostgresRollback(t, dburl, func(atx tldb.Adapter) { fn(t, atx) })
	})
}

// TestStatRegistrationConsistency guards against drift between three lists
// that must stay aligned: statTables (stat-name → tables), AllStats (write
// order), and dmfr.GetFeedVersionTables().FetchStatDerivedTables (canonical
// stat-derived table list). A missing entry in any of them causes silent
// skips at runtime.
func TestStatRegistrationConsistency(t *testing.T) {
	gotTables := map[string]bool{}
	for _, ts := range statTables {
		for _, table := range ts {
			gotTables[table] = true
		}
	}
	for _, table := range dmfr.GetFeedVersionTables().FetchStatDerivedTables {
		assert.True(t, gotTables[table], "table %q is in FetchStatDerivedTables but missing from statTables", table)
	}
	assert.Equal(t, len(dmfr.GetFeedVersionTables().FetchStatDerivedTables), len(gotTables), "statTables covers a different number of tables than FetchStatDerivedTables")

	for _, stat := range AllStats {
		_, ok := statTables[stat]
		assert.True(t, ok, "AllStats entry %q has no statTables entry", stat)
	}
	assert.Equal(t, len(AllStats), len(statTables), "AllStats and statTables have different lengths")
}

func TestValidateStatNames(t *testing.T) {
	cases := []struct {
		name    string
		stats   []string
		wantErr bool
	}{
		{name: "empty means all", stats: nil, wantErr: false},
		{name: "all valid names", stats: AllStats, wantErr: false},
		{name: "single invalid", stats: []string{"badname"}, wantErr: true},
		{name: "mix valid and invalid", stats: []string{StatGeohash, "badname"}, wantErr: true},
		{name: "case sensitive", stats: []string{"Geohash"}, wantErr: true},
		{name: "empty string is invalid", stats: []string{""}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateStatNames(tc.stats)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
