package stats

import (
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/stretchr/testify/assert"
)

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
