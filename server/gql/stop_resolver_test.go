package gql

import (
	"context"
	"fmt"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/gql/internal/gqlcase"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestStopResolver(t *testing.T) {
	c, cfg := newTestClient(t)
	gqlcase.QueryTestcases(t, c, stopResolverTestcases(t, cfg))
}

func BenchmarkStopResolver(b *testing.B) {
	c, cfg := newTestClient(b)
	gqlcase.BenchmarkTestcases(b, c, stopResolverTestcases(b, cfg))
}

// Onestop ID lookups read the materialized table's own onestop_id when the server
// uses materialized tables, and the onestop ID history otherwise. Both must agree.
func TestStopResolver_OnestopID_Materialized(t *testing.T) {
	testcases := []testcase{
		{
			Name:         "where onestop_id",
			Query:        `query{stops(where:{onestop_id:"s-9q9k658fd1-sanjosediridoncaltrain"}) {stop_id} }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"70262"},
		},
		{
			Name:         "where onestop_ids",
			Query:        `query{stops(where:{onestop_ids:["s-9q9k658fd1-sanjosediridoncaltrain","s-9q9p1wxf72-macarthur"]}) {stop_id} }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"70262", "MCAR", "MCAR_S"},
		},
		{
			// Two records share MacArthur's onestop_id.
			Name:         "selected onestop_id",
			Query:        `query{stops(where:{onestop_id:"s-9q9p1wxf72-macarthur"}) {stop_id onestop_id} }`,
			Selector:     "stops.#.onestop_id",
			SelectExpect: []string{"s-9q9p1wxf72-macarthur", "s-9q9p1wxf72-macarthur"},
		},
		{
			Name:         "unknown onestop_id",
			Query:        `query{stops(where:{onestop_id:"s-0000000000-nowhere"}) {stop_id} }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{},
		},
		// With previous IDs allowed, a current ID is answered without the history,
		// and anything else by it. Fruitvale's onestop_id changed between versions;
		// either way the stop comes back with its current one.
		{
			Name:         "previous allowed, current id",
			Query:        `query{stops(where:{onestop_id:"s-9q9nfsxn67-fruitvale", allow_previous_onestop_ids:true}) {stop_id} }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"FTVL"},
		},
		{
			Name:  "previous allowed, previous id",
			Query: `query{stops(where:{onestop_id:"s-9q9nfswzpg-fruitvale", allow_previous_onestop_ids:true}) {stop_id onestop_id} }`,
			Sel: []testcaseSelector{
				{Selector: "stops.#.stop_id", Expect: []string{"FTVL"}},
				{Selector: "stops.#.onestop_id", Expect: []string{"s-9q9nfsxn67-fruitvale"}},
			},
		},
		{
			Name:         "previous allowed, current and previous ids",
			Query:        `query{stops(where:{onestop_ids:["s-9q9k658fd1-sanjosediridoncaltrain","s-9q9nfswzpg-fruitvale"], allow_previous_onestop_ids:true}) {stop_id} }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"70262", "FTVL"},
		},
		{
			Name:         "previous allowed, current and previous ids of one stop",
			Query:        `query{stops(where:{onestop_ids:["s-9q9nfsxn67-fruitvale","s-9q9nfswzpg-fruitvale"], allow_previous_onestop_ids:true}) {stop_id} }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"FTVL"},
		},
		{
			Name:         "previous allowed, unknown id",
			Query:        `query{stops(where:{onestop_id:"s-0000000000-nowhere", allow_previous_onestop_ids:true}) {stop_id} }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{},
		},
	}
	for _, materialized := range []bool{false, true} {
		t.Run(fmt.Sprintf("UseMaterialized=%t", materialized), func(t *testing.T) {
			c, _ := newTestClientWithOpts(t, testconfig.Options{UseMaterialized: materialized})
			gqlcase.QueryTestcases(t, c, testcases)
		})
	}
}

// Stops found with previous IDs allowed page like any others.
func TestStopResolver_OnestopID_PreviousCursor(t *testing.T) {
	c, cfg := newTestClient(t)
	osid := "s-9q9p1wxf72-macarthur"
	ents, err := cfg.Finder.FindStops(context.Background(), nil, nil, nil, &model.StopFilter{OnestopID: &osid})
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 2 {
		t.Fatalf("got %d stops for %s, expected 2", len(ents), osid)
	}
	query := `query($after:Int){stops(after:$after, limit:1, where:{onestop_id:"s-9q9p1wxf72-macarthur", allow_previous_onestop_ids:true}) {stop_id} }`
	gqlcase.QueryTestcases(t, c, []testcase{
		{
			Name:         "first page",
			Query:        query,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{ents[0].StopID.Val},
		},
		{
			Name:         "second page",
			Query:        query,
			Vars:         hw{"after": ents[0].ID},
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{ents[1].StopID.Val},
		},
	})
}

// With previous IDs allowed, a current match that the rest of the filter excludes
// leaves the ID to the history. Here the old BART version gives Fruitvale the
// onestop ID San Jose Diridon has now.
func TestStopResolver_OnestopID_PreviousFiltered(t *testing.T) {
	sanJose := "s-9q9k658fd1-sanjosediridoncaltrain"
	testcases := []testcase{
		{
			Name:         "current match",
			Query:        `query($osid:String!){stops(where:{onestop_id:$osid, allow_previous_onestop_ids:true}) {stop_id} }`,
			Vars:         hw{"osid": sanJose},
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"70262"},
		},
		{
			Name:         "current match filtered out",
			Query:        `query($osid:String!){stops(where:{onestop_id:$osid, allow_previous_onestop_ids:true, feed_onestop_id:"BA"}) {stop_id} }`,
			Vars:         hw{"osid": sanJose},
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"FTVL"},
		},
	}
	for _, materialized := range []bool{false, true} {
		t.Run(fmt.Sprintf("UseMaterialized=%t", materialized), func(t *testing.T) {
			testconfig.ConfigTxRollback(t, testconfig.Options{UseMaterialized: materialized}, func(cfg model.Config) {
				ctx := context.Background()
				sha1 := "dd7aca4a8e4c90908fd3603c097fabee75fea907"
				fvs, err := cfg.Finder.FindFeedVersions(ctx, nil, nil, nil, &model.FeedVersionFilter{Sha1: &sha1})
				if err != nil || len(fvs) != 1 {
					t.Fatalf("feed version %s: %v", sha1, err)
				}
				previous := stats.FeedVersionStats{StopOnestopIDs: []dmfr.FeedVersionStopOnestopID{{EntityID: "FTVL", OnestopID: sanJose}}}
				if err := stats.WriteFeedVersionStats(ctx, cfg.Adapter, previous, fvs[0].ID, stats.WriteOptions{Stats: []string{stats.StatOnestopIDs}}); err != nil {
					t.Fatal(err)
				}
				srv := model.AddConfigAndPerms(cfg, NewDefaultHandler())
				gqlcase.QueryTestcases(t, client.New(usercheck.UserDefaultMiddleware("test")(srv)), testcases)
			})
		})
	}
}

func TestStopResolverLocation(t *testing.T) {
	c, cfg := newTestClient(t)
	gqlcase.QueryTestcases(t, c, stopResolverLocationTestcases(t, cfg))
}

func TestStopResolver_PreviousOnestopID(t *testing.T) {
	c, cfg := newTestClient(t)
	gqlcase.QueryTestcases(t, c, stopResolverPreviousOnestopIDTestcases(t, cfg))
}

func TestStopResolver_AdminCache(t *testing.T) {
	type canLoadAdmins interface {
		LoadAdmins(context.Context) error
	}
	c, cfg := newTestClient(t)
	if v, ok := cfg.Finder.(canLoadAdmins); !ok {
		t.Fatal("finder cant load admins")
	} else {
		if err := v.LoadAdmins(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	q := `query($feed_version_sha1:String!, $stop_id:String!) { stops(where:{stop_id:$stop_id, feed_version_sha1:$feed_version_sha1}) { place { adm0_name adm1_name adm0_iso adm1_iso } } }`
	tcs := []testcase{
		{
			Name:         "usa",
			Query:        q,
			Vars:         hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "stop_id": "FTVL"},
			Selector:     "stops.#.place.adm0_name",
			SelectExpect: []string{"United States of America"},
		},
		{
			Name:         "california",
			Query:        q,
			Vars:         hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "stop_id": "FTVL"},
			Selector:     "stops.#.place.adm1_name",
			SelectExpect: []string{"California"},
		},
		{
			Name:         "adm0_iso",
			Query:        q,
			Vars:         hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "stop_id": "FTVL"},
			Selector:     "stops.#.place.adm0_iso",
			SelectExpect: []string{"US"},
		},
		{
			Name:         "adm1_iso",
			Query:        q,
			Vars:         hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "stop_id": "FTVL"},
			Selector:     "stops.#.place.adm1_iso",
			SelectExpect: []string{"US-CA"},
		},
		{
			Name:         "florida",
			Query:        q,
			Vars:         hw{"feed_version_sha1": "c969427f56d3a645195dd8365cde6d7feae7e99b", "stop_id": "8032"},
			Selector:     "stops.#.place.adm1_name",
			SelectExpect: []string{"Florida"},
		},
	}
	gqlcase.QueryTestcases(t, c, tcs)
}

func stopResolverTestcases(t testing.TB, cfg model.Config) []testcase {
	stopObsFvid := 0
	if err := cfg.Adapter.DBX().QueryRowx("select feed_version_id from ext_performance_stop_observations limit 1").Scan(&stopObsFvid); err != nil {
		t.Errorf("could not get fvid for stop observation test: %s", err.Error())
	}
	testcases := []testcase{
		// census_geographies: see per-stop buffer cases below
		// stop external references
		{
			Name: "external reference: working",
			Query: `query {
				stops(where:{feed_onestop_id: "BA", stop_id:"FTVL"}) {
				  external_reference {
					target_stop_id
					target_feed_onestop_id
					target_active_stop {
					  stop_id
					}
				  }
				}
			  }`,
			F: func(t *testing.T, jj string) {
				assert.EqualValues(t, "CT", gjson.Get(jj, "stops.0.external_reference.target_feed_onestop_id").String())
				assert.EqualValues(t, "70041", gjson.Get(jj, "stops.0.external_reference.target_stop_id").String())
				assert.EqualValues(t, "70041", gjson.Get(jj, "stops.0.external_reference.target_active_stop.stop_id").String())
			},
		},
		{
			Name: "external reference: broken",
			Query: `query {
				stops(where:{feed_onestop_id: "BA", stop_id:"POWL"}) {
				  external_reference {
					target_stop_id
					target_feed_onestop_id
					target_active_stop {
					  stop_id
					}
				  }
				}
			  }`,
			F: func(t *testing.T, jj string) {
				assert.EqualValues(t, "CT", gjson.Get(jj, "stops.0.external_reference.target_feed_onestop_id").String())
				assert.EqualValues(t, "missing", gjson.Get(jj, "stops.0.external_reference.target_stop_id").String())
				assert.EqualValues(t, nil, gjson.Get(jj, "stops.0.external_reference.target_active_stop").Value())
			},
		},
		// stop observations
		{
			Name: "observations: found",
			Query: `query($fvid:Int!,$day:Date!) {
				stops(where:{feed_onestop_id: "BA", stop_id:"FTVL"}) {
					observations(where:{feed_version_id:$fvid, source:"TripUpdate", trip_start_date:$day}) {
						trip_id
						route_id
						observed_arrival_time
						observed_departure_time
					}
				}
			  }`,
			Vars: hw{"fvid": stopObsFvid, "day": "2023-03-09"},
			F: func(t *testing.T, jj string) {
				assert.EqualValues(t, "test", gjson.Get(jj, "stops.0.observations.0.trip_id").String())
				assert.EqualValues(t, "03", gjson.Get(jj, "stops.0.observations.0.route_id").String())
				assert.EqualValues(t, "10:00:00", gjson.Get(jj, "stops.0.observations.0.observed_arrival_time").String())
				assert.EqualValues(t, "10:00:10", gjson.Get(jj, "stops.0.observations.0.observed_departure_time").String())
				assert.EqualValues(t, 1, len(gjson.Get(jj, "stops.0.observations").Array()))
			},
		},
		{
			Name: "observations: none",
			Query: `query($fvid:Int!,$day:Date!) {
				stops(where:{feed_onestop_id: "BA", stop_id:"FTVL"}) {
					observations(where:{feed_version_id:$fvid, source:"TripUpdate", trip_start_date:$day}) {
						trip_id
						route_id
						observed_arrival_time
						observed_departure_time
					}
				}
			  }`,
			Vars:         hw{"fvid": stopObsFvid, "day": "2023-03-08"},
			Selector:     "stops.0.observations.#.trip_id",
			SelectExpect: []string{},
		},
		// census_geographies (per-stop buffer attribution)
		{
			Name:         "census_geographies tract by stop buffer",
			Query:        `query{stops(where:{feed_onestop_id:"BA", stop_id:"FTVL"}) { stop_id census_geographies(where:{layer:"tract", radius:100.0}) { geoid } } }`,
			Selector:     "stops.0.census_geographies.#.geoid",
			SelectExpect: []string{"1400000US06001406100"},
		},
		{
			Name:         "census_geographies county by stop buffer",
			Query:        `query{stops(where:{feed_onestop_id:"BA", stop_id:"MCAR"}) { stop_id census_geographies(where:{layer:"county", radius:1000.0}) { geoid } } }`,
			Selector:     "stops.0.census_geographies.#.geoid",
			SelectExpect: []string{"0500000US06001"},
		},
		{
			// Two stops resolved in one request batch into a single
			// CensusGeographiesByEntityIDs call; each must get its own
			// geographies (including the tracts shared by both buffers).
			// Before per-stop attribution every row carried match_entity_id=0
			// and both stops resolved to empty lists.
			Name: "census_geographies per-stop attribution across batch",
			Query: `query{
				s12: stops(where:{feed_onestop_id:"BA", stop_id:"12TH"}) { stop_id census_geographies(where:{layer:"tract", radius:300.0}) { geoid } }
				s19: stops(where:{feed_onestop_id:"BA", stop_id:"19TH"}) { stop_id census_geographies(where:{layer:"tract", radius:300.0}) { geoid } }
			}`,
			Sel: []testcaseSelector{
				{
					Selector: "s12.0.census_geographies.#.geoid",
					Expect:   []string{"1400000US06001402801", "1400000US06001402802", "1400000US06001402900", "1400000US06001403000", "1400000US06001403100"},
				},
				{
					Selector: "s19.0.census_geographies.#.geoid",
					Expect:   []string{"1400000US06001402801", "1400000US06001402900"},
				},
			},
		},
	}
	return testcases
}

func stopResolverLocationTestcases(t *testing.T, cfg model.Config) []testcase {
	geographyId := 0
	if err := cfg.Adapter.DBX().QueryRowx(`select id from tl_census_geographies where geoid = '1400000US06001402900'`).Scan(&geographyId); err != nil {
		t.Errorf("could not get geography id for test: %s", err.Error())
	}

	testcases := []testcase{
		// within geography ids
		{
			Name:         "where within geography ids 1",
			Query:        `query($geographyIds:[Int!]){stops(where:{location:{geography_ids:$geographyIds}}){id stop_id}}`,
			Selector:     "stops.#.stop_id",
			Vars:         hw{"geographyIds": []int{geographyId}},
			SelectExpect: []string{"19TH", "19TH_N"},
		},
	}
	return testcases

}

func stopResolverPreviousOnestopIDTestcases(t testing.TB, cfg model.Config) []testcase {
	_ = t
	_ = cfg
	testcases := []testcase{
		{
			Name:         "default",
			Query:        `query($osid:String!, $previous:Boolean!) { stops(where:{onestop_id:$osid, allow_previous_onestop_ids:$previous}) { stop_id onestop_id }}`,
			Vars:         hw{"osid": "s-9q9nfsxn67-fruitvale", "previous": false},
			Selector:     "stops.#.onestop_id",
			SelectExpect: []string{"s-9q9nfsxn67-fruitvale"},
		},
		{
			Name:         "old id no result",
			Query:        `query($osid:String!, $previous:Boolean!) { stops(where:{onestop_id:$osid, allow_previous_onestop_ids:$previous}) { stop_id onestop_id }}`,
			Vars:         hw{"osid": "s-9q9nfswzpg-fruitvale", "previous": false},
			Selector:     "stops.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:         "old id no specify fv",
			Query:        `query($osid:String!, $previous:Boolean!) { stops(where:{onestop_id:$osid, allow_previous_onestop_ids:$previous, feed_version_sha1:"dd7aca4a8e4c90908fd3603c097fabee75fea907"}) { stop_id onestop_id }}`,
			Vars:         hw{"osid": "s-9q9nfswzpg-fruitvale", "previous": false},
			Selector:     "stops.#.onestop_id",
			SelectExpect: []string{"s-9q9nfswzpg-fruitvale"},
		},
		{
			Name:         "use previous",
			Query:        `query($osid:String!, $previous:Boolean!) { stops(where:{onestop_id:$osid, allow_previous_onestop_ids:$previous}) { stop_id onestop_id }}`,
			Vars:         hw{"osid": "s-9q9nfswzpg-fruitvale", "previous": true},
			Selector:     "stops.#.onestop_id",
			SelectExpect: []string{"s-9q9nfsxn67-fruitvale"},
		},
	}
	return testcases
}
