package dbfinder

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/stretchr/testify/require"
)

// When two census sources share a layer, the batched loader must return that layer
// for both. Deduping the geography->layer join on layer id alone dropped a shared
// layer from every source but one.
func TestCensusSourceLayersBySourceIDs_SharedLayer(t *testing.T) {
	ctx := context.Background()
	db := testutil.MustOpenTestDB(t)
	dbf := NewFinder(dbutil.WithQueryLogger(db, false, 0))

	// Both sources have geographies in the tract layer -- the shared-layer case that
	// exposes the batch collision (a single source id would dedup correctly).
	var s1, s2 int
	if err := db.Get(&s1, "select id from tl_census_sources where name = 'tl_2024_06_tract.zip'"); err != nil {
		t.Skipf("census fixture not loaded: %v", err)
	}
	require.NoError(t, db.Get(&s2, "select id from tl_census_sources where name = 'tl_2024_53_tract.zip'"))

	got, errs := dbf.CensusSourceLayersBySourceIDs(ctx, []int{s1, s2})
	for _, err := range errs {
		require.NoError(t, err)
	}
	require.Len(t, got, 2)
	require.Contains(t, layerNames(got[0]), "tract", "first source missing shared layer")
	require.Contains(t, layerNames(got[1]), "tract", "second source missing shared layer")
}

func layerNames(layers []*model.CensusLayer) []string {
	names := make([]string, 0, len(layers))
	for _, l := range layers {
		names = append(names, l.Name)
	}
	return names
}
