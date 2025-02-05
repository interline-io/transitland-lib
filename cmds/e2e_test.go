package cmds

// End to end tests for sync, fetch, and import

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/fetch"
	"github.com/interline-io/transitland-lib/importer"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/stretchr/testify/assert"
)

func TestE2E(t *testing.T) {
	tcs := []struct {
		name              string
		fn                string
		activate          bool
		fvcount           int
		unimport          bool
		unimportSchedOnly bool
		expectStops       int
		expectRoutes      int
		expectStopTimes   int
	}{
		{
			name:            "basic",
			fn:              "testdata/example.zip",
			activate:        true,
			fvcount:         1,
			expectStops:     9,
			expectRoutes:    5,
			expectStopTimes: 28,
		},
		{
			name:            "basic no activate",
			fn:              "testdata/example.zip",
			activate:        false,
			fvcount:         1,
			expectStops:     0,
			expectRoutes:    0,
			expectStopTimes: 0,
		},
		{
			name:            "basic unimport",
			fn:              "testdata/example.zip",
			activate:        true,
			unimport:        true,
			fvcount:         1,
			expectStops:     0,
			expectRoutes:    0,
			expectStopTimes: 0,
		},
		{
			name:              "basic unimport sched",
			fn:                "testdata/example.zip",
			activate:          true,
			unimport:          true,
			unimportSchedOnly: true,
			fvcount:           1,
			expectStops:       9,
			expectRoutes:      5,
			expectStopTimes:   0,
		},
		{
			name:            "basic nested dir",
			fn:              "testdata/example-nested-dir.zip#example-nested-dir/example",
			activate:        true,
			fvcount:         1,
			expectStops:     9,
			expectRoutes:    5,
			expectStopTimes: 28,
		},
		{
			name:            "basic nested two feeds 1",
			fn:              "testdata/example-nested-two-feeds.zip#example1",
			activate:        true,
			fvcount:         1,
			expectStops:     9,
			expectRoutes:    1,
			expectStopTimes: 4,
		},

		{
			name:            "basic nested two feeds 2",
			fn:              "testdata/example-nested-two-feeds.zip#example2",
			activate:        true,
			fvcount:         1,
			expectStops:     9,
			expectRoutes:    5,
			expectStopTimes: 28,
		},
		{
			name:            "basic nested zip",
			fn:              "testdata/example-nested-zip.zip#example-nested-zip/example.zip",
			activate:        true,
			fvcount:         1,
			expectStops:     9,
			expectRoutes:    5,
			expectStopTimes: 28,
		},
	}
	ctx := context.TODO()
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				p := strings.Split(tc.fn, "#")
				buf, err := ioutil.ReadFile(testpath.RelPath(p[0]))
				if err != nil {
					t.Error(err)
				}
				w.Write(buf)
			}))
			defer ts.Close()

			tmpdir, err := ioutil.TempDir("", "gtfs")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpdir) // clean up

			atx := testdb.TempSqliteAdapter()

			// Fetch
			feedName := tc.name
			fetch := FetchCommand{
				CreateFeed: true,
				FeedIDs:    []string{feedName},
				Workers:    1,
				Adapter:    atx,
				Options: fetch.Options{
					FeedURL:   ts.URL + "/" + tc.fn,
					Storage:   tmpdir,
					FetchedAt: time.Now(),
				},
			}
			if err := fetch.Run(ctx); err != nil {
				t.Fatal(err)
			}

			// Import
			impcmd := ImportCommand{
				FeedIDs: []string{feedName},
				Workers: 1,
				Adapter: atx,
				Options: importer.Options{
					Storage:  tmpdir,
					Activate: tc.activate,
				},
			}
			if err := impcmd.Run(ctx); err != nil {
				t.Fatal(err)
			}

			// Unimport
			fvid := 0
			testdb.ShouldGet(t, atx, &fvid, "select id from feed_versions order by id desc limit 1")
			if tc.unimport {
				unimpcmd := UnimportCommand{
					FVIDs:        []string{strconv.Itoa(fvid)},
					ScheduleOnly: tc.unimportSchedOnly,
					Workers:      1,
					Adapter:      atx,
				}
				if err := unimpcmd.Run(ctx); err != nil {
					t.Fatal(err)
				}
			}

			// Test
			fvcount := 0
			testdb.ShouldGet(t, atx, &fvcount, "SELECT count(*) FROM feed_versions fv JOIN current_feeds cf on cf.id = fv.feed_id WHERE cf.onestop_id = ?", feedName)
			assert.Equal(t, tc.fvcount, fvcount, "feed_version count")
			scount := 0
			testdb.ShouldGet(t, atx, &scount, "SELECT count(*) FROM gtfs_stops JOIN feed_states fs using(feed_version_id)")
			assert.Equal(t, tc.expectStops, scount, "stop count")
			rcount := 0
			testdb.ShouldGet(t, atx, &rcount, "SELECT count(*) FROM gtfs_routes JOIN feed_states fs using(feed_version_id)")
			assert.Equal(t, tc.expectRoutes, rcount, "route count")
			stcount := 0
			testdb.ShouldGet(t, atx, &stcount, "SELECT count(*) FROM gtfs_stop_times JOIN feed_states fs using(feed_version_id)")
			assert.Equal(t, tc.expectStopTimes, stcount, "stop time count")

		})
	}
}
