package test

// Separate test package because we need to import "actions" into "testconfig"

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/gbfs"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/finders/actions"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/testdata"
	sq "github.com/irees/squirrel"
	"github.com/stretchr/testify/assert"
	"github.com/twpayne/go-geom"
)

type testWorker struct {
	kind  string
	count *int64
}

func (t *testWorker) Kind() string {
	return t.kind
}

func (t *testWorker) Run(ctx context.Context) error {
	time.Sleep(1 * time.Millisecond)
	atomic.AddInt64(t.count, 1)
	return nil
}

func TestGbfsFetch(t *testing.T) {
	ts := httptest.NewServer(gbfs.NewTestGbfsServer("en", testdata.Path("server/gbfs")))
	defer ts.Close()
	testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
		ctx := model.WithConfig(context.Background(), cfg)
		if err := actions.GbfsFetch(ctx, "test-gbfs", ts.URL+"/gbfs.json"); err != nil {
			t.Fatal(err)
		}

		// Test
		bikes, err := cfg.GbfsFinder.FindBikes(
			ctx,
			nil,
			&model.GbfsBikeRequest{
				Near: &model.PointRadius{
					Lon:    -122.396445,
					Lat:    37.793250,
					Radius: 100,
				},
			},
		)
		if err != nil {
			t.Fatal(err)
		}
		bikeids := []string{}
		for _, ent := range bikes {
			bikeids = append(bikeids, ent.BikeID.Val)
		}
		assert.ElementsMatch(t, []string{"2e09a0ed99c8ad32cca516661618645e"}, bikeids)
	})
}

func TestStaticFetchWorker(t *testing.T) {
	tcs := []struct {
		name               string
		feedId             string
		serveFile          string
		expectError        bool
		expectResponseCode int64
		expectResponseSize int64
		expectResponseSHA1 string
		expectSuccess      bool
	}{
		{
			name:               "bart existing",
			feedId:             "BA",
			serveFile:          "server/gtfs/bart.zip",
			expectResponseCode: 200,
			expectResponseSize: 456139,
			expectResponseSHA1: "e535eb2b3b9ac3ef15d82c56575e914575e732e0",
			expectSuccess:      true,
		},
		{
			name:               "bart existing old",
			feedId:             "BA",
			serveFile:          "server/gtfs/bart-old.zip",
			expectResponseCode: 200,
			expectResponseSize: 429721,
			expectResponseSHA1: "dd7aca4a8e4c90908fd3603c097fabee75fea907",
			expectSuccess:      true,
		},
		{
			name:               "bart invalid",
			feedId:             "BA",
			serveFile:          "server/gtfs/invalid.zip",
			expectResponseCode: 200,
			expectResponseSize: 12,
			expectResponseSHA1: "88af471a23dfdc103e67752dd56128ae77b8debe",
			expectError:        false,
			expectSuccess:      false,
		},
		{
			name:               "bart new",
			feedId:             "BA",
			serveFile:          "server/gtfs/bart-new.zip",
			expectResponseCode: 200,
			expectResponseSize: 1151609,
			expectResponseSHA1: "b40aa01814bf92dba06dbccdebcc3aefa6208248",
			expectError:        false,
			expectSuccess:      true,
		},
		{
			name:               "hart existing",
			feedId:             "HA",
			serveFile:          "server/gtfs/hart.zip",
			expectResponseCode: 200,
			expectResponseSize: 3543136,
			expectResponseSHA1: "c969427f56d3a645195dd8365cde6d7feae7e99b",
			expectSuccess:      true,
		},
		{
			name:               "404",
			feedId:             "BA",
			serveFile:          "example.zip",
			expectError:        false,
			expectResponseCode: 404,
			expectSuccess:      false,
		},
		{
			name:        "invalid feed",
			feedId:      "unknown",
			serveFile:   "example.zip",
			expectError: true,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Setup http
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/"+tc.serveFile {
					http.Error(w, "404", 404)
					return

				}
				buf, err := os.ReadFile(testdata.Path(tc.serveFile))
				if err != nil {
					http.Error(w, "404", 404)
					return
				}
				w.Write(buf)
			}))
			defer ts.Close()

			// Setup job
			feedUrl := ts.URL + "/" + tc.serveFile
			testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
				cfg.Checker = nil // disable checker for this test
				ctx := model.WithConfig(context.Background(), cfg)
				// Run job
				if result, err := actions.StaticFetch(ctx, tc.feedId, nil, feedUrl); err != nil && !tc.expectError {
					_ = result
					t.Fatal("unexpected error", err)
				} else if err == nil && tc.expectError {
					t.Fatal("expected responseError")
				} else if err != nil && tc.expectError {
					return
				}
				// Check output
				ff := dmfr.FeedFetch{}
				if err := dbutil.Get(
					ctx,
					cfg.Finder.DBX(),
					sq.StatementBuilder.
						Select("ff.*").
						From("feed_fetches ff").
						Join("current_feeds cf on cf.id = ff.feed_id").
						Where(sq.Eq{"cf.onestop_id": tc.feedId}).
						Where(sq.Eq{"ff.url": feedUrl}).
						OrderBy("ff.id desc").
						Limit(1),
					&ff,
				); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, tc.expectResponseCode, ff.ResponseCode.Val, "expect response_code")
				assert.Equal(t, tc.expectSuccess, ff.Success, "expect success")
				assert.Equal(t, tc.expectResponseSize, ff.ResponseSize.Val, "expect response_size")
				if tc.expectResponseSHA1 != "" {
					assert.Equal(t, tc.expectResponseSHA1, ff.ResponseSHA1.Val, "expect response_sha1")
				}
			})

		})
	}
}

func TestValidateUpload(t *testing.T) {
	tcs := []struct {
		name        string
		serveFile   string
		rtUrls      []string
		expectError bool
		f           func(*testing.T, *model.ValidationReport)
	}{
		{
			name:      "ct",
			serveFile: "server/gtfs/caltrain.zip",
			rtUrls:    []string{"rt/CT-vp-error.json"},
			f: func(t *testing.T, result *model.ValidationReport) {
				if len(result.Errors) != 1 {
					t.Fatal("expected errors")
					return
				}
				if len(result.Errors[0].Errors) != 1 {
					t.Fatal("expected errors")
					return
				}
				g := result.Errors[0].Errors[0]
				if v, ok := g.Geometry.Val.(*geom.GeometryCollection); ok {
					ggs := v.Geoms()
					assert.Equal(t, len(ggs), 2)
					assert.Equal(t, len(ggs[0].FlatCoords()), 1112)
					assert.Equal(t, len(ggs[1].FlatCoords()), 2)
				}
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			// Setup http
			ts := testutil.NewTestServer(testdata.Path())
			defer ts.Close()

			// Setup job
			testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
				cfg.Checker = nil // disable checker for this test
				ctx := model.WithConfig(context.Background(), cfg)
				// Run job
				feedUrl := ts.URL + "/" + tc.serveFile
				var rturls []string
				for _, v := range tc.rtUrls {
					rturls = append(rturls, ts.URL+"/"+v)
				}
				result, err := actions.ValidateUpload(ctx, nil, &feedUrl, rturls)
				if err != nil && !tc.expectError {
					_ = result
					t.Fatal("unexpected error", err)
				} else if err == nil && tc.expectError {
					t.Fatal("expected responseError")
				} else if err != nil && tc.expectError {
					return
				}
				if tc.f != nil {
					tc.f(t, result)
				}
				// jj, _ := json.MarshalIndent(result, "", "  ")
				// fmt.Println(string(jj))
			})
		})
	}
}
