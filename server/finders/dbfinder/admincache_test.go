package dbfinder

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/tlxy"
	sq "github.com/irees/squirrel"
	"github.com/stretchr/testify/assert"
)

type testCase struct {
	name       string
	point      tlxy.Point
	expectAdm0 string
	expectAdm1 string
	expectOk   bool
	skipPg     bool
}

// Note: these values are based on the Natural Earth 10m data set, which is slightly simplified. For instance, the Georgia/Florida boundary used below.
func getTestCases() []testCase {
	tcs := []testCase{
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
	return tcs
}

func TestAdminCache(t *testing.T) {
	dbx := testutil.MustOpenTestDB(t)
	c, err := newAdminCache(context.Background(), dbx)
	if err != nil {
		t.Fatal(err)
	}
	tcs := getTestCases()
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			r, ok := c.Check(tc.point)
			assert.Equal(t, tc.expectAdm0, r.Adm0Name)
			assert.Equal(t, tc.expectAdm1, r.Adm1Name)
			assert.Equal(t, tc.expectOk, ok)
			if tc.skipPg {
				return
			}
			var pgCheck []struct {
				Name  string
				Admin string
			}
			q := sq.
				Select("ne.name", "ne.admin", "ne.geometry").
				From("ne_10m_admin_1_states_provinces ne").
				Where("ST_Intersects(ne.geometry::geography, ST_MakePoint(?,?)::geography)", tc.point.Lon, tc.point.Lat)
			if err := dbutil.Select(context.Background(), dbx, q, &pgCheck); err != nil {
				t.Fatal(err)
			}
			// if len(pgCheck) != tc.expectOk {
			// 	t.Error("expectOk did not match result from postgres")
			// }
			for _, ent := range pgCheck {
				assert.Equal(t, tc.expectAdm0, ent.Admin, "different than postgres")
				assert.Equal(t, tc.expectAdm1, ent.Name, "different than postgres")
			}
		})
	}
}

func BenchmarkTestAdminCache(b *testing.B) {
	dbx := testutil.MustOpenTestDB(b)
	c, err := newAdminCache(context.Background(), dbx)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	tcs := getTestCases()
	for _, tc := range tcs {
		b.Run(tc.name, func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				r, ok := c.Check(tc.point)
				_ = ok
				_ = r
			}
		})
	}
}

func BenchmarkTestAdminCache_LoadAdmins(b *testing.B) {
	dbx := testutil.MustOpenTestDB(b)
	c := &adminCache{}
	for n := 0; n < b.N; n++ {
		if err := c.loadAdmins(context.Background(), dbx); err != nil {
			b.Fatal(err)
		}
	}
}
