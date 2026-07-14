package importer

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/feedmanager"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testreader"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	if dburl == "" {
		log.Infof("TL_TEST_DATABASE_URL is not set, skipping")
		return
	}
	os.Exit(m.Run())
}

// setupFetch creates a feed version and the stats a fetch writes for it -- file infos, service
// levels, onestop ids -- without importing it.
func setupFetch(ctx context.Context, t *testing.T, atx tldb.Adapter) int {
	fv := testdb.CreateTestFeedVersion(atx, testreader.ExampleZip.URL)
	tlreader, err := tlcsv.NewReader(testreader.ExampleZip.URL)
	if err != nil {
		t.Fatal(err)
	}
	if err := stats.CreateFeedStats(ctx, atx, tlreader, fv.ID, stats.WriteOptions{}); err != nil {
		t.Fatal(err)
	}
	return fv.ID
}

func setupImport(ctx context.Context, t *testing.T, atx tldb.Adapter) int {
	fvid := setupFetch(ctx, t, atx)
	if _, err := ImportFeedVersion(ctx, feedmanager.NewDBFeedManager(atx), Options{FeedVersionID: fvid, Storage: "/"}); err != nil {
		t.Fatal(err)
	}
	return fvid
}

// countFV counts a feed version's rows in a table.
func countFV(t *testing.T, atx tldb.Adapter, table string, fvid int) int {
	count := 0
	if err := atx.Sqrl().Select("count(*)").From(table).Where(sq.Eq{"feed_version_id": fvid}).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

func softDeleted(t *testing.T, atx tldb.Adapter, fvid int) bool {
	count := 0
	testdb.ShouldGet(t, atx, &count, "SELECT count(*) FROM feed_versions WHERE id = ? AND deleted_at IS NOT NULL", fvid)
	return count > 0
}

// Most feed versions are fetched and never imported, and delete is what reclaims them: their
// fetch-time rows are the bulk of the tables this is meant to keep from growing without bound.
func TestDeleteFeedVersion_NeverImported(t *testing.T) {
	ctx := context.TODO()
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
		fvid := setupFetch(ctx, t, atx)

		// The fetch really did write the rows we are about to claim were removed.
		if countFV(t, atx, "feed_version_stop_onestop_ids", fvid) == 0 {
			t.Fatal("expected the fetch to have written stop onestop ids")
		}

		if err := DeleteFeedVersion(ctx, atx, fvid, nil); err != nil {
			t.Fatal(err)
		}
		for _, table := range []string{
			"feed_version_file_infos",
			"feed_version_service_levels",
			"feed_version_stop_onestop_ids",
			"feed_version_route_onestop_ids",
			"feed_version_agency_onestop_ids",
			"tl_feed_version_geohashes",
		} {
			assert.Equal(t, 0, countFV(t, atx, table, fvid), table)
		}
		assert.True(t, softDeleted(t, atx, fvid), "feed version should be soft deleted")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUnimportFeedVersion(t *testing.T) {
	ctx := context.TODO()
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
		fvid := setupImport(ctx, t, atx)
		// TODO: test ExtraTables option
		if err := UnimportFeedVersion(ctx, atx, fvid, UnimportOptions{}); err != nil {
			t.Fatal(err)
		}
		tcs := []struct {
			table  string
			expect int
		}{
			{
				table:  "gtfs_stops",
				expect: 0,
			},
			{
				table:  "gtfs_trips",
				expect: 0,
			},
			{
				table:  "gtfs_stop_times",
				expect: 0,
			},
			{
				table:  "feed_version_stop_onestop_ids",
				expect: 9,
			},
			{
				table:  "tl_feed_version_geometries",
				expect: 0,
			},
			{
				table:  "feed_version_gtfs_imports",
				expect: 0,
			},
		}
		for _, tc := range tcs {
			t.Run(tc.table, func(t *testing.T) {
				count := 0
				if err := atx.Sqrl().Select("count(*)").From(tc.table).Where(sq.Eq{"feed_version_id": fvid}).Scan(&count); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, tc.expect, count, tc.table)
			})
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

// Unimport refuses a feed version whose import is in flight: the copier commits as it goes, so
// deleting under it would remove rows the import has already written, and the import would then
// finalize success = true over the hole. Every other record state -- and a missing record, whose
// rows nothing else can reach -- stays unimportable.
func TestUnimportFeedVersion_ImportInProgress(t *testing.T) {
	ctx := context.TODO()
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	tcs := []struct {
		name       string
		record     bool
		success    bool
		inProgress bool
		refused    bool
	}{
		{"imported", true, true, false, false},
		{"import in flight", true, false, true, true},
		{"import failed", true, false, false, false},
		{"unimport interrupted", true, true, true, false},
		{"no import record", false, false, false, false},
	}
	err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				fv := testdb.CreateTestFeedVersion(atx, "test.zip")
				if tc.record {
					testdb.CreateTestFeedVersionImport(atx, fv.ID, tc.success, tc.inProgress)
				}
				// One entity row, so a refusal has something to leave alone.
				stop := gtfs.Stop{
					StopID:       tt.NewString("s"),
					Geometry:     tt.NewPoint(-122, 37),
					LocationType: tt.NewInt(0),
				}
				stop.FeedVersionID = fv.ID
				testdb.MustInsert(atx, &stop)

				err := UnimportFeedVersion(ctx, atx, fv.ID, UnimportOptions{})
				if tc.refused {
					if !errors.Is(err, ErrImportInProgress) {
						t.Fatalf("expected ErrImportInProgress, got %v", err)
					}
					if countFV(t, atx, "gtfs_stops", fv.ID) == 0 {
						t.Error("a refused unimport must not delete anything")
					}
					// Force is how an import that died mid-run gets cleaned up.
					if err := UnimportFeedVersion(ctx, atx, fv.ID, UnimportOptions{Force: true}); err != nil {
						t.Fatal(err)
					}
				} else if err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, 0, countFV(t, atx, "gtfs_stops", fv.ID), "gtfs_stops")
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// An unimport hides the feed version before deleting anything, and the deletes are the slow part,
// so an interrupted unimport is ordinary. Retrying one must always be allowed -- including for a
// failed import, which is the state the import command tells the operator to unimport.
//
// The trap: flagging in_progress on a failed record would produce (success=false,
// in_progress=true), which is exactly "import in flight", the one state unimport refuses. The
// retry would then be refused forever, and only --force could reach it.
func TestUnimportFeedVersion_InterruptedUnimportCanRetry(t *testing.T) {
	ctx := context.TODO()
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	for _, success := range []bool{true, false} {
		name := "imported"
		if !success {
			name = "import failed"
		}
		t.Run(name, func(t *testing.T) {
			err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
				fv := testdb.CreateTestFeedVersion(atx, "test.zip")
				testdb.CreateTestFeedVersionImport(atx, fv.ID, success, false)

				// Start an unimport and interrupt it: the flag commits, the deletes do not run.
				if err := setImportInProgress(ctx, atx, fv.ID); err != nil {
					t.Fatal(err)
				}
				if err := UnimportFeedVersion(ctx, atx, fv.ID, UnimportOptions{}); err != nil {
					t.Fatalf("retrying an interrupted unimport must be allowed, got %v", err)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Delete refuses any feed version that still has an import record, whatever state it is in: the
// record may belong to an import in flight, whose rows the copier is still writing.
func TestDeleteFeedVersion_RequiresUnimport(t *testing.T) {
	ctx := context.TODO()
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	tcs := []struct {
		name       string
		success    bool
		inProgress bool
	}{
		{"imported", true, false},
		{"import in flight", false, true},
		{"import failed", false, false},
		{"unimport interrupted", true, true},
	}
	err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
		for _, tc := range tcs {
			t.Run(tc.name, func(t *testing.T) {
				// Delete refuses before reading an entity row, so the record is the whole fixture.
				fv := testdb.CreateTestFeedVersion(atx, "test.zip")
				testdb.CreateTestFeedVersionImport(atx, fv.ID, tc.success, tc.inProgress)
				if err := DeleteFeedVersion(ctx, atx, fv.ID, nil); !errors.Is(err, ErrFeedVersionImported) {
					t.Fatalf("expected ErrFeedVersionImported, got %v", err)
				}
				if softDeleted(t, atx, fv.ID) {
					t.Error("refused delete should not have soft deleted the feed version")
				}
			})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// Entity rows can outlive their import record -- a crash, or something deleting the record
// directly. Delete is terminal, so it must not leave them behind: the missing record is only a
// proxy for "not imported", and here the proxy is wrong.
func TestDeleteFeedVersion_OrphanedRows(t *testing.T) {
	ctx := context.TODO()
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
		fvid := setupImport(ctx, t, atx)
		// Lose the import record, keeping the entity rows.
		if _, err := atx.Sqrl().Delete("feed_version_gtfs_imports").
			Where(sq.Eq{"feed_version_id": fvid}).ExecContext(ctx); err != nil {
			t.Fatal(err)
		}
		if countFV(t, atx, "gtfs_stops", fvid) == 0 {
			t.Fatal("expected orphaned stops to exist")
		}

		if err := DeleteFeedVersion(ctx, atx, fvid, nil); err != nil {
			t.Fatal(err)
		}
		for _, table := range []string{"gtfs_stops", "gtfs_trips", "gtfs_stop_times", "feed_version_stop_onestop_ids"} {
			assert.Equal(t, 0, countFV(t, atx, table, fvid), table)
		}
		assert.True(t, softDeleted(t, atx, fvid), "feed version should be soft deleted")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteFeedVersion(t *testing.T) {
	ctx := context.TODO()
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
		fvid := setupImport(ctx, t, atx)
		// Unimport is a precondition, not something delete does for you.
		if err := UnimportFeedVersion(ctx, atx, fvid, UnimportOptions{}); err != nil {
			t.Fatal(err)
		}
		if err := DeleteFeedVersion(ctx, atx, fvid, nil); err != nil {
			t.Fatal(err)
		}
		for _, table := range []string{
			"gtfs_stops",
			"gtfs_trips",
			"gtfs_stop_times",
			"feed_version_stop_onestop_ids",
			"feed_version_gtfs_imports",
			"tl_feed_version_geometries",
		} {
			assert.Equal(t, 0, countFV(t, atx, table, fvid), table)
		}
		assert.True(t, softDeleted(t, atx, fvid), "feed version should be soft deleted")
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}
