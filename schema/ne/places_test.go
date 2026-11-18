package ne

import (
	"context"
	"maps"
	"path"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/ext/builders"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

// placeIndex reads the embedded data once for the package's tests.
var placeIndex = sync.OnceValues(PlaceIndex)

func testPlaceIndex(t testing.TB) *builders.PlaceIndex {
	t.Helper()
	ix, err := placeIndex()
	if err != nil {
		t.Fatal(err)
	}
	return ix
}

func TestPlaceIndex_NearestAdmin(t *testing.T) {
	ix := testPlaceIndex(t)
	// Natural Earth's 10m boundaries are slightly simplified, as at the
	// Georgia/Florida border here. Postgres matches only the points inside an
	// area, so it doesn't check the one point offshore.
	tcs := []struct {
		name       string
		point      tlxy.Point
		expectAdm0 string
		expectAdm1 string
		expectOk   bool
		skipPg     bool
	}{
		{name: "new york", expectAdm0: "United States of America", expectAdm1: "New York", expectOk: true, point: tlxy.Point{Lon: -74.132285, Lat: 40.625665}},
		{name: "california", expectAdm0: "United States of America", expectAdm1: "California", expectOk: true, point: tlxy.Point{Lon: -122.431297, Lat: 37.773972}},
		{name: "kansas 1", expectAdm0: "United States of America", expectAdm1: "Kansas", expectOk: true, point: tlxy.Point{Lon: -98.85867269364557, Lat: 39.96773433000109}},
		{name: "kansas 2", expectAdm0: "United States of America", expectAdm1: "Kansas", expectOk: true, point: tlxy.Point{Lon: -98.85867269364557, Lat: 39.99901}},
		{name: "nebraska 1", expectAdm0: "United States of America", expectAdm1: "Nebraska", expectOk: true, point: tlxy.Point{Lon: -98.862255, Lat: 40.001587}},
		{name: "nebraska 2", expectAdm0: "United States of America", expectAdm1: "Nebraska", expectOk: true, point: tlxy.Point{Lon: -98.867745, Lat: 40.003185}},
		{name: "utah", expectAdm0: "United States of America", expectAdm1: "Utah", expectOk: true, point: tlxy.Point{Lon: -109.056664, Lat: 40.996479}},
		{name: "colorado", expectAdm0: "United States of America", expectAdm1: "Colorado", expectOk: true, point: tlxy.Point{Lon: -109.045685, Lat: 40.997833}},
		{name: "wyoming", expectAdm0: "United States of America", expectAdm1: "Wyoming", expectOk: true, point: tlxy.Point{Lon: -109.050133, Lat: 41.002209}},
		{name: "north dakota", expectAdm0: "United States of America", expectAdm1: "North Dakota", expectOk: true, point: tlxy.Point{Lon: -100.964531, Lat: 45.946934}},
		{name: "georgia", expectAdm0: "United States of America", expectAdm1: "Georgia", expectOk: true, point: tlxy.Point{Lon: -82.066697, Lat: 30.370054}},
		{name: "florida", expectAdm0: "United States of America", expectAdm1: "Florida", expectOk: true, point: tlxy.Point{Lon: -82.046522, Lat: 30.360419}},
		{name: "saskatchewan", expectAdm0: "Canada", expectAdm1: "Saskatchewan", expectOk: true, point: tlxy.Point{Lon: -102.007904, Lat: 58.269615}},
		{name: "manitoba", expectAdm0: "Canada", expectAdm1: "Manitoba", expectOk: true, point: tlxy.Point{Lon: -101.982025, Lat: 58.269245}},
		{name: "paris", expectAdm0: "France", expectAdm1: "Paris", expectOk: true, point: tlxy.Point{Lon: 2.4729377, Lat: 48.8589143}},
		{name: "texas", expectAdm0: "United States of America", expectAdm1: "Texas", expectOk: true, point: tlxy.Point{Lon: -94.794261, Lat: 29.289210}},
		{name: "texas water 1", skipPg: true, expectAdm0: "United States of America", expectAdm1: "Texas", expectOk: true, point: tlxy.Point{Lon: -94.784667, Lat: 29.286234}},
		{name: "texas water 2", expectOk: false, point: tlxy.Point{Lon: -94.237, Lat: 26.874}},
		{name: "null", expectOk: false, point: tlxy.Point{Lon: 0, Lat: 0}},
	}
	_, hasDB := testutil.CheckTestDB()
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			a, ok := ix.NearestAdmin(tc.point)
			assert.Equal(t, tc.expectOk, ok)
			assert.Equal(t, tc.expectAdm0, a.Admin)
			assert.Equal(t, tc.expectAdm1, a.Name)
			if tc.skipPg || !hasDB {
				return
			}
			var pgCheck []struct {
				Name  string
				Admin string
			}
			q := `select ne.name, ne.admin from ne_10m_admin_1_states_provinces ne where ST_Intersects(ne.geometry::geography, ST_MakePoint($1, $2)::geography)`
			if err := testutil.MustOpenTestDB(t).Select(&pgCheck, q, tc.point.Lon, tc.point.Lat); err != nil {
				t.Fatal(err)
			}
			for _, ent := range pgCheck {
				assert.Equal(t, tc.expectAdm0, ent.Admin, "different than postgres")
				assert.Equal(t, tc.expectAdm1, ent.Name, "different than postgres")
			}
		})
	}
}

// The test server database's agency places came from its own Natural Earth
// tables, which the embedded data loaded. Built in memory, each feed version's
// places should come out the same, with ranks close to the database's: Postgres
// measures distance on the spheroid, so a stop nearly equidistant from two
// places can go to the other one.
func TestPlaceIndex_AgencyPlaces(t *testing.T) {
	if msg, ok := testutil.CheckTestDB(); !ok {
		t.Skip(msg)
	}
	ix := testPlaceIndex(t)
	type placeKey struct {
		AgencyID string    `db:"agency_id"`
		Name     tt.String `db:"name"`
		Adm1name tt.String `db:"adm1name"`
		Adm0name tt.String `db:"adm0name"`
	}
	var rows []struct {
		URL string `db:"url"`
		placeKey
		Rank float64 `db:"rank"`
	}
	q := `select fv.url, a.agency_id, p.name, p.adm1name, p.adm0name, p.rank
	from tl_agency_places p
	join gtfs_agencies a on a.id = p.agency_id
	join feed_versions fv on fv.id = p.feed_version_id
	order by fv.url`
	if err := testutil.MustOpenTestDB(t).Select(&rows, q); err != nil {
		t.Fatal(err)
	}
	if len(rows) == 0 {
		t.Fatal("the test database has no agency places")
	}
	byURL := map[string]map[placeKey]float64{}
	var urls []string
	for _, r := range rows {
		if _, ok := byURL[r.URL]; !ok {
			urls = append(urls, r.URL)
			byURL[r.URL] = map[placeKey]float64{}
		}
		byURL[r.URL][r.placeKey] = r.Rank
	}
	for _, url := range urls {
		want := byURL[url]
		t.Run(path.Base(url), func(t *testing.T) {
			// The versions' urls name files under testdata, from the repo root.
			zip := strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(url, "file://"), "./"), "testdata/")
			reader, err := tlcsv.NewReader(testdata.Path(zip))
			if err != nil {
				t.Fatal(err)
			}
			opts := copier.Options{}
			opts.AddExtension(builders.NewAgencyPlaceBuilder(ix))
			w := direct.NewWriter()
			if _, err := copier.CopyWithOptions(context.Background(), reader, w, opts); err != nil {
				t.Fatal(err)
			}
			got := map[placeKey]float64{}
			for _, ent := range w.Reader.OtherList {
				if ap, ok := ent.(*builders.AgencyPlace); ok {
					got[placeKey{AgencyID: ap.AgencyID, Name: ap.Name, Adm1name: ap.Adm1name, Adm0name: ap.Adm0name}] = ap.Rank
				}
			}
			assert.ElementsMatch(t, slices.Collect(maps.Keys(want)), slices.Collect(maps.Keys(got)))
			for k, rank := range want {
				if g, ok := got[k]; ok {
					assert.InDelta(t, rank, g, 0.01, "%s %s, %s", k.Name.Val, k.Adm1name.Val, k.Adm0name.Val)
				}
			}
		})
	}
}
