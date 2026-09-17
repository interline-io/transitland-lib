package dbfinder

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	sq "github.com/irees/squirrel"
	"github.com/stretchr/testify/assert"
)

// Exercises placeSearchSelect against stand-in association rows, for names no
// fixture feed has. Each row's n stands in for the operator count: the query
// orders by it after the search's own ordering.
func TestPlaceSearchSelect(t *testing.T) {
	dbx := testutil.MustOpenTestDB(t)
	type row struct {
		name, adm1, adm0 string
		n                int
	}
	adm0 := model.PlaceAggregationLevelAdm0
	adm0Adm1 := model.PlaceAggregationLevelAdm0Adm1
	city := model.PlaceAggregationLevelAdm0Adm1City
	tcs := []struct {
		name   string
		level  model.PlaceAggregationLevel
		rows   []row
		search string
		expect []string
	}{
		{
			name: "accented name, unaccented search", level: city, search: "zurich",
			rows:   []row{{"Zürich", "Zürich", "Switzerland", 1}},
			expect: []string{"Zürich, Zürich"},
		},
		{
			name: "accented name, accented search", level: city, search: "Zürich",
			rows:   []row{{"Zürich", "Zürich", "Switzerland", 1}},
			expect: []string{"Zürich, Zürich"},
		},
		{
			name: "every word of an accented name", level: city, search: "sao paulo",
			rows:   []row{{"São Paulo", "São Paulo", "Brazil", 1}},
			expect: []string{"São Paulo, São Paulo"},
		},
		{
			name: "hyphenated name as written", level: city, search: "winston-salem",
			rows:   []row{{"Winston-Salem", "North Carolina", "United States of America", 1}},
			expect: []string{"Winston-Salem, North Carolina"},
		},
		{
			name: "name with an apostrophe as written", level: city, search: "coeur d'alene",
			rows:   []row{{"Coeur d'Alene", "Idaho", "United States of America", 1}},
			expect: []string{"Coeur d'Alene, Idaho"},
		},
		{
			name: "region qualifies a city", level: city, search: "portland maine",
			rows: []row{
				{"Portland", "Oregon", "United States of America", 2},
				{"Portland", "Maine", "United States of America", 1},
			},
			expect: []string{"Portland, Maine"},
		},
		{
			name: "region alone is not a city", level: city, search: "california",
			rows:   []row{{"Oakland", "California", "United States of America", 1}},
			expect: []string{},
		},
		{
			name: "own-name matches on every word first", level: city, search: "new york",
			rows: []row{
				{"Newburgh", "New York", "United States of America", 2},
				{"New York", "New York", "United States of America", 1},
			},
			expect: []string{"New York, New York", "Newburgh, New York"},
		},
		{
			name: "then by operator count", level: city, search: "new",
			rows: []row{
				{"Newburgh", "New York", "United States of America", 2},
				{"New York", "New York", "United States of America", 1},
			},
			expect: []string{"Newburgh, New York", "New York, New York"},
		},
		{
			name: "region search needs its own name", level: adm0Adm1, search: "georgia",
			rows: []row{
				{"", "Georgia", "United States of America", 1},
				{"", "Tbilisi", "Georgia", 1},
			},
			expect: []string{"Georgia"},
		},
		{
			name: "accented country", level: adm0, search: "curacao",
			rows:   []row{{"", "", "Curaçao", 1}},
			expect: []string{"Curaçao"},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			rows := sq.StatementBuilder.Select()
			for i, r := range tc.rows {
				if i == 0 {
					rows = rows.
						Column("?::text as name", r.name).
						Column("?::text as adm1name", r.adm1).
						Column("?::text as adm0name", r.adm0).
						Column("?::int as n", r.n)
				} else {
					rows = rows.Suffix("union all select ?::text, ?::text, ?::text, ?::int", r.name, r.adm1, r.adm0, r.n)
				}
			}
			q := sq.StatementBuilder.Select().FromSelect(rows, "tlap")
			switch tc.level {
			case adm0:
				q = q.Column("tlap.adm0name as label").GroupBy("tlap.adm0name")
			case adm0Adm1:
				q = q.Column("tlap.adm1name as label").GroupBy("tlap.adm0name", "tlap.adm1name")
			default:
				q = q.Column("concat_ws(', ', tlap.name, tlap.adm1name) as label").GroupBy("tlap.adm0name", "tlap.adm1name", "tlap.name")
			}
			q = placeSearchSelect(q, &tc.level, tc.search).OrderBy("max(tlap.n) desc")

			var got []struct {
				Label string `db:"label"`
			}
			if err := dbutil.Select(context.Background(), dbx, q, &got); err != nil {
				t.Fatal(err)
			}
			labels := []string{}
			for _, g := range got {
				labels = append(labels, g.Label)
			}
			assert.Equal(t, tc.expect, labels)
		})
	}
}

// Search orders places by the operators their count reports, not by agencies.
// With Caltrain's agency resolved to BART's operator, San Francisco and San Mateo
// each have two agencies but one operator, tying with San Jose.
func TestPlaceSelectSearchOperatorOrder(t *testing.T) {
	ctx := context.Background()
	tx, err := testutil.MustOpenTestDB(t).BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	merge := sq.StatementBuilder.
		Update("current_operators_in_feed").
		Set("resolved_onestop_id", "o-9q9-bayarearapidtransit").
		Where(sq.Eq{"resolved_onestop_id": "o-9q9-caltrain"})
	if err := dbutil.Update(ctx, tx, merge); err != nil {
		t.Fatal(err)
	}

	level := model.PlaceAggregationLevelAdm0Adm1City
	search := "san"
	q := placeSelect(nil, nil, nil, &level, nil, &model.PlaceFilter{Search: &search}, model.PlaceGeometrySelect{})
	var got []*model.Place
	if err := dbutil.Select(ctx, tx, q, &got); err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, p := range got {
		names = append(names, *p.CityName)
	}
	assert.Equal(t, []string{"San Francisco", "San Jose", "San Mateo"}, names)
}
