package stats

import (
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/stretchr/testify/assert"
)

// TestStatTablesCoverFetchStatDerived guards against drift between statTables
// (the WriteOptions stat-name → table mapping) and the canonical list in
// dmfr.GetFeedVersionTables().FetchStatDerivedTables. If a new stat-derived
// table is added without a corresponding statTables entry, the default
// (no --stats flag) write path would silently skip deleting it on rebuild.
func TestStatTablesCoverFetchStatDerived(t *testing.T) {
	got := map[string]bool{}
	for _, ts := range statTables {
		for _, table := range ts {
			got[table] = true
		}
	}
	expected := dmfr.GetFeedVersionTables().FetchStatDerivedTables
	for _, table := range expected {
		assert.True(t, got[table], "table %q is in FetchStatDerivedTables but missing from statTables", table)
	}
	assert.Equal(t, len(expected), len(got), "statTables covers a different number of tables than FetchStatDerivedTables")
}

// TestAllStatsMatchesStatTables ensures every constant in AllStats has a
// corresponding entry in statTables and vice versa.
func TestAllStatsMatchesStatTables(t *testing.T) {
	for _, stat := range AllStats {
		_, ok := statTables[stat]
		assert.True(t, ok, "AllStats entry %q has no statTables entry", stat)
	}
	assert.Equal(t, len(AllStats), len(statTables), "AllStats and statTables have different lengths")
}

func TestWriteOptions_Validate(t *testing.T) {
	cases := []struct {
		name    string
		stats   []string
		wantErr bool
	}{
		{name: "empty means all", stats: nil, wantErr: false},
		{name: "empty slice means all", stats: []string{}, wantErr: false},
		{name: "single valid", stats: []string{StatGeohash}, wantErr: false},
		{name: "multiple valid", stats: []string{StatGeohash, StatOnestopIDs, StatServiceLevels}, wantErr: false},
		{name: "all valid names", stats: AllStats, wantErr: false},
		{name: "single invalid", stats: []string{"badname"}, wantErr: true},
		{name: "mix valid and invalid", stats: []string{StatGeohash, "badname"}, wantErr: true},
		{name: "case sensitive", stats: []string{"Geohash"}, wantErr: true},
		{name: "empty string is invalid", stats: []string{""}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := WriteOptions{Stats: tc.stats}.Validate()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWriteOptions_resolveStats_defaults(t *testing.T) {
	enabled, err := WriteOptions{}.resolveStats()
	assert.NoError(t, err)
	for _, stat := range AllStats {
		assert.True(t, enabled[stat], "stat %q should be enabled when WriteOptions.Stats is empty", stat)
	}
	assert.Equal(t, len(AllStats), len(enabled))
}

func TestWriteOptions_resolveStats_subset(t *testing.T) {
	enabled, err := WriteOptions{Stats: []string{StatGeohash}}.resolveStats()
	assert.NoError(t, err)
	assert.True(t, enabled[StatGeohash])
	assert.False(t, enabled[StatFileInfos])
	assert.False(t, enabled[StatServiceLevels])
	assert.False(t, enabled[StatServiceWindows])
	assert.False(t, enabled[StatOnestopIDs])
}

func TestWriteOptions_Validate_errorMessage(t *testing.T) {
	err := WriteOptions{Stats: []string{"badname"}}.Validate()
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "badname")
		assert.Contains(t, err.Error(), StatGeohash, "error should list valid stat names")
	}
}
