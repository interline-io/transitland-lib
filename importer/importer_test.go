package importer

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/feedmanager"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/interline-io/transitland-lib/internal/testreader"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
)

func TestImportFeedVersion(t *testing.T) {
	ctx := context.TODO()
	setup := func(atx tldb.Adapter, filename string) int {
		// Create FV
		fv := dmfr.FeedVersion{}
		fv.EarliestCalendarDate = tt.NewDate(time.Now())
		fv.LatestCalendarDate = tt.NewDate(time.Now())
		fv.File = filename
		return testdb.ShouldInsert(t, atx, &fv)
	}
	t.Run("Success", func(t *testing.T) {
		testdb.TempSqlite(func(atx tldb.Adapter) error {
			fvid := setup(atx, testreader.ExampleZip.URL)
			atx2 := testdb.AdapterIgnoreTx{Adapter: atx}
			_, err := ImportFeedVersion(ctx, feedmanager.NewDBFeedManager(&atx2), Options{Activate: true, FeedVersionID: fvid, Storage: "/"})
			if err != nil {
				t.Fatal(err)
			}
			// Check results
			fvi := dmfr.FeedVersionImport{}
			testdb.ShouldGet(t, atx, &fvi, "SELECT * FROM feed_version_gtfs_imports WHERE feed_version_id = ?", fvid)
			if fvi.Success != true {
				t.Errorf("expected success = true")
			}
			if fvi.ExceptionLog != "" {
				t.Errorf("expected empty, got %s", fvi.ExceptionLog)
			}
			if fvi.InProgress != false {
				t.Errorf("expected in_progress = false")
			}
			// An import with no ImportSource set defaults to automatic.
			if fvi.ImportSource != dmfr.ImportSourceAutomatic {
				t.Errorf("expected import_source = %q, got %q", dmfr.ImportSourceAutomatic, fvi.ImportSource)
			}
			count := 0
			expstops := testreader.ExampleZip.Counts["stops.txt"]
			testdb.ShouldGet(t, atx, &count, "SELECT count(*) FROM gtfs_stops WHERE feed_version_id = ?", fvid)
			if count != expstops {
				t.Errorf("got %d stops, expect %d stops", count, expstops)
			}
			expfvistops := fvi.EntityCount["stops.txt"]
			if count != expfvistops {
				t.Errorf("got %d stops, expect %d stops", count, expfvistops)
			}
			return nil
		})
	})
	t.Run("ManualSource", func(t *testing.T) {
		testdb.TempSqlite(func(atx tldb.Adapter) error {
			fvid := setup(atx, testreader.ExampleZip.URL)
			atx2 := testdb.AdapterIgnoreTx{Adapter: atx}
			_, err := ImportFeedVersion(ctx, feedmanager.NewDBFeedManager(&atx2), Options{FeedVersionID: fvid, Storage: "/", ImportSource: dmfr.ImportSourceManual})
			if err != nil {
				t.Fatal(err)
			}
			fvi := dmfr.FeedVersionImport{}
			testdb.ShouldGet(t, atx, &fvi, "SELECT * FROM feed_version_gtfs_imports WHERE feed_version_id = ?", fvid)
			if fvi.ImportSource != dmfr.ImportSourceManual {
				t.Errorf("expected import_source = %q, got %q", dmfr.ImportSourceManual, fvi.ImportSource)
			}
			return nil
		})
	})
	t.Run("Failed", func(t *testing.T) {
		fvid := 0
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			fvid = setup(atx, testpath.RelPath("testdata/gtfs-examples/does-not-exist"))
			atx2 := testdb.AdapterIgnoreTx{Adapter: atx}
			_, err := ImportFeedVersion(ctx, feedmanager.NewDBFeedManager(&atx2), Options{FeedVersionID: fvid, Storage: "/"})
			if err == nil {
				t.Errorf("expected an error, got none")
			}
			fvi := dmfr.FeedVersionImport{}
			testdb.ShouldGet(t, atx, &fvi, "SELECT * FROM feed_version_gtfs_imports WHERE feed_version_id = ?", fvid)
			if fvi.Success != false {
				t.Errorf("expected success = false")
			}
			if fvi.ExceptionLog == "" {
				t.Error("got no exception log error, expected something", fvi.ExceptionLog)
			}
			if fvi.InProgress != false {
				t.Errorf("expected in_progress = false")
			}
			return nil
		})
		if err != nil {
			t.Error(err)
		}
	})
}

// A failed import leaves the copied rows in place -- not rolled back, kept out of entity queries
// only by the failed import record. TestImportFeedVersion/Failed can't cover this: it fails on a
// missing file, before the copier writes anything.
func TestImportFeedVersion_FailedImportLeavesRows(t *testing.T) {
	ctx := context.TODO()
	// Not TempSqlite: it runs inside a transaction, the thing this import no longer has.
	atx := testdb.TempSqliteAdapter()
	fv := testdb.CreateTestFeedVersion(atx, testreader.ExampleZip.URL)

	// A negative threshold can never be met and is checked after the copy, so the import fails
	// with every entity already written.
	if _, err := ImportFeedVersion(ctx, feedmanager.NewDBFeedManager(atx), Options{
		FeedVersionID:  fv.ID,
		Storage:        "/",
		ErrorThreshold: map[string]float64{"*": -1},
	}); err == nil {
		t.Fatal("expected the import to fail on the error threshold")
	}

	// Entity queries gate on success and in_progress, so check them as stored.
	fvi := dmfr.FeedVersionImport{}
	testdb.ShouldGet(t, atx, &fvi, "SELECT * FROM feed_version_gtfs_imports WHERE feed_version_id = ?", fv.ID)
	if fvi.Success || fvi.InProgress {
		t.Errorf("stored import record is success=%v in_progress=%v, want false/false", fvi.Success, fvi.InProgress)
	}

	count := 0
	testdb.ShouldGet(t, atx, &count, "SELECT count(*) FROM gtfs_stops WHERE feed_version_id = ?", fv.ID)
	if count == 0 {
		t.Error("expected the failed import to leave its stops behind")
	}
}

// A cancelled caller ctx (commonly a client disconnect) must not strand the import record
// in_progress: the finalize writes run under context.WithoutCancel so they still commit.
func TestImportFeedVersion_CancelledContextStillFinalizes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	atx := testdb.TempSqliteAdapter()
	fv := testdb.CreateTestFeedVersion(atx, testreader.ExampleZip.URL)

	// Cancel once the record exists, so ctx is already canceled when the import finishes the
	// writer and saves the record.
	fm := &cancelOnCreateFeedManager{
		writerLifecycleFeedManager: &writerLifecycleFeedManager{DBFeedManager: feedmanager.NewDBFeedManager(atx)},
		cancel:                     cancel,
	}
	if _, err := ImportFeedVersion(ctx, fm, Options{FeedVersionID: fv.ID, Storage: "/"}); err != nil {
		t.Fatalf("import should finalize under a cancelled context, got: %v", err)
	}

	// Finalized out of in_progress despite the cancelled ctx: WithoutCancel let the write commit.
	fvi := dmfr.FeedVersionImport{}
	testdb.ShouldGet(t, atx, &fvi, "SELECT * FROM feed_version_gtfs_imports WHERE feed_version_id = ?", fv.ID)
	if fvi.InProgress || !fvi.Success {
		t.Errorf("stored record is success=%v in_progress=%v, want true/false; finalize did not survive cancellation", fvi.Success, fvi.InProgress)
	}
}

// cancelOnCreateFeedManager cancels the import context the moment the import record is created,
// standing in for a client that disconnects mid-import.
type cancelOnCreateFeedManager struct {
	*writerLifecycleFeedManager
	cancel context.CancelFunc
}

func (m *cancelOnCreateFeedManager) CreateFeedVersionImport(ctx context.Context, fvi *dmfr.FeedVersionImport) (int, error) {
	id, err := m.writerLifecycleFeedManager.CreateFeedVersionImport(ctx, fvi)
	m.cancel()
	return id, err
}

// An import ends the entity writer it opens with exactly one of Finish and Abort, before it saves
// the import's record.
func TestImportFeedVersion_EntityWriter(t *testing.T) {
	ctx := context.TODO()
	importWith := func(t *testing.T, fm *writerLifecycleFeedManager, opts Options) (*dmfr.FeedVersionImport, error) {
		t.Helper()
		atx := testdb.TempSqliteAdapter()
		fv := testdb.CreateTestFeedVersion(atx, testreader.ExampleZip.URL)
		fm.DBFeedManager = feedmanager.NewDBFeedManager(atx)
		opts.FeedVersionID = fv.ID
		opts.Storage = "/"
		result, importErr := ImportFeedVersion(ctx, fm, opts)
		stored, err := fm.GetFeedVersionImport(ctx, fv.ID)
		if err != nil {
			t.Fatal(err)
		}
		if got := result.FeedVersionImport; stored != nil && (got.Success != stored.Success || got.InProgress != stored.InProgress) {
			t.Errorf("import returned success=%v in_progress=%v, but stored success=%v in_progress=%v", got.Success, got.InProgress, stored.Success, stored.InProgress)
		}
		return stored, importErr
	}
	checkCalls := func(t *testing.T, fm *writerLifecycleFeedManager, want ...string) {
		t.Helper()
		if !slices.Equal(fm.calls, want) {
			t.Errorf("got calls %v, want %v", fm.calls, want)
		}
	}
	t.Run("Success", func(t *testing.T) {
		fm := &writerLifecycleFeedManager{}
		stored, err := importWith(t, fm, Options{})
		if err != nil {
			t.Fatal(err)
		}
		checkCalls(t, fm, "open", "create", "finish", "save")
		if !stored.Success || fm.finished.ID != stored.ID || !fm.finished.Success || fm.finished.InProgress {
			t.Errorf("Finish got record %d success=%v in_progress=%v, want the saved record %d with success", fm.finished.ID, fm.finished.Success, fm.finished.InProgress, stored.ID)
		}
		want := testreader.ExampleZip.Counts["stops.txt"]
		if got := fm.finished.EntityCount["stops.txt"]; got != want {
			t.Errorf("Finish got a record with %d stops, want %d", got, want)
		}
		if fm.finishedStops != want {
			t.Errorf("the finished writer received %d stops, want %d", fm.finishedStops, want)
		}
	})
	t.Run("FailedCreate", func(t *testing.T) {
		createErr := errors.New("create failed")
		abortErr := errors.New("abort failed")
		fm := &writerLifecycleFeedManager{createErr: createErr, abortErr: abortErr}
		stored, err := importWith(t, fm, Options{})
		if !errors.Is(err, createErr) || !errors.Is(err, abortErr) {
			t.Fatalf("got error %v, want both %v and %v", err, createErr, abortErr)
		}
		checkCalls(t, fm, "open", "create", "abort")
		if stored != nil {
			t.Errorf("got import record %d, want none", stored.ID)
		}
	})
	t.Run("FailedCopy", func(t *testing.T) {
		// A negative threshold can never be met, so the import fails after the copy.
		abortErr := errors.New("abort failed")
		fm := &writerLifecycleFeedManager{abortErr: abortErr}
		stored, err := importWith(t, fm, Options{ErrorThreshold: map[string]float64{"*": -1}})
		if !errors.Is(err, abortErr) {
			t.Fatalf("got error %v, want one that includes %v", err, abortErr)
		}
		checkCalls(t, fm, "open", "create", "abort", "save")
		if stored.Success || stored.InProgress || stored.ExceptionLog != err.Error() {
			t.Errorf("stored record is success=%v in_progress=%v log=%q, want a failed record with the import's error", stored.Success, stored.InProgress, stored.ExceptionLog)
		}
	})
	t.Run("FailedFinish", func(t *testing.T) {
		finishErr := errors.New("commit failed")
		fm := &writerLifecycleFeedManager{finishErr: finishErr}
		stored, err := importWith(t, fm, Options{})
		if !errors.Is(err, finishErr) {
			t.Fatalf("got error %v, want one that includes %v", err, finishErr)
		}
		checkCalls(t, fm, "open", "create", "finish", "save")
		if stored.Success || stored.InProgress || stored.ExceptionLog != err.Error() {
			t.Errorf("stored record is success=%v in_progress=%v log=%q, want a failed record with the import's error", stored.Success, stored.InProgress, stored.ExceptionLog)
		}
	})
	t.Run("FailedSave", func(t *testing.T) {
		// The failed record doesn't save, so the stored record stays in progress.
		finishErr := errors.New("commit failed")
		saveErr := errors.New("save failed")
		fm := &writerLifecycleFeedManager{finishErr: finishErr, saveErr: saveErr}
		stored, err := importWith(t, fm, Options{})
		if !errors.Is(err, finishErr) || !errors.Is(err, saveErr) {
			t.Fatalf("got error %v, want both %v and %v", err, finishErr, saveErr)
		}
		checkCalls(t, fm, "open", "create", "finish", "save")
		if stored.Success || !stored.InProgress {
			t.Errorf("stored record is success=%v in_progress=%v, want false/true", stored.Success, stored.InProgress)
		}
	})
	t.Run("FailedOpen", func(t *testing.T) {
		openErr := errors.New("open failed")
		fm := &writerLifecycleFeedManager{openErr: openErr}
		stored, err := importWith(t, fm, Options{})
		if !errors.Is(err, openErr) {
			t.Fatalf("got error %v, want %v", err, openErr)
		}
		checkCalls(t, fm, "open")
		if stored != nil {
			t.Errorf("got import record %d, want none for a writer that didn't open", stored.ID)
		}
	})
}

// writerLifecycleFeedManager records, in order, the import's calls that open and end its entity
// writer and create and save its record, and fails the steps a test asks it to fail.
type writerLifecycleFeedManager struct {
	*feedmanager.DBFeedManager
	openErr, createErr, finishErr, abortErr, saveErr error // what each step returns

	calls         []string               // open, create, finish, abort and save, as the import called each
	finished      dmfr.FeedVersionImport // the record Finish got
	finishedStops int                    // the stops the finished writer received
}

func (m *writerLifecycleFeedManager) EntityWriter(ctx context.Context, fvid int) (feedmanager.EntityWriter, error) {
	m.calls = append(m.calls, "open")
	if m.openErr != nil {
		return nil, m.openErr
	}
	w, err := m.DBFeedManager.EntityWriter(ctx, fvid)
	if err != nil {
		return nil, err
	}
	return &lifecycleWriter{EntityWriter: w, fm: m}, nil
}

func (m *writerLifecycleFeedManager) CreateFeedVersionImport(ctx context.Context, fvi *dmfr.FeedVersionImport) (int, error) {
	m.calls = append(m.calls, "create")
	if m.createErr != nil {
		return 0, m.createErr
	}
	return m.DBFeedManager.CreateFeedVersionImport(ctx, fvi)
}

func (m *writerLifecycleFeedManager) UpdateFeedVersionImport(ctx context.Context, fvi *dmfr.FeedVersionImport) error {
	m.calls = append(m.calls, "save")
	if m.saveErr != nil {
		return m.saveErr
	}
	return m.DBFeedManager.UpdateFeedVersionImport(ctx, fvi)
}

// lifecycleWriter counts the stops it receives, and records the import's Finish or Abort.
type lifecycleWriter struct {
	feedmanager.EntityWriter
	fm    *writerLifecycleFeedManager
	stops int
}

func (w *lifecycleWriter) AddEntities(ents []tt.Entity) ([]string, error) {
	for _, ent := range ents {
		if ent.Filename() == "stops.txt" {
			w.stops++
		}
	}
	return w.EntityWriter.AddEntities(ents)
}

func (w *lifecycleWriter) Finish(ctx context.Context, fvi dmfr.FeedVersionImport) error {
	w.fm.calls = append(w.fm.calls, "finish")
	w.fm.finished = fvi
	w.fm.finishedStops = w.stops
	// Fail under a canceled ctx, as a commit through a database would.
	if err := ctx.Err(); err != nil {
		return err
	}
	return w.fm.finishErr
}

func (w *lifecycleWriter) Abort(ctx context.Context) error {
	w.fm.calls = append(w.fm.calls, "abort")
	return w.fm.abortErr
}

func Test_importFeedVersion(t *testing.T) {
	ctx := context.TODO()
	err := testdb.TempSqlite(func(atx tldb.Adapter) error {
		// Create FV
		fv := dmfr.FeedVersion{File: testreader.ExampleZip.URL}
		fv.EarliestCalendarDate = tt.NewDate(time.Now())
		fv.LatestCalendarDate = tt.NewDate(time.Now())
		fvid := testdb.ShouldInsert(t, atx, &fv)
		fv.ID = fvid // TODO: ?? Should be set by canSetID
		// Import
		fm := feedmanager.NewDBFeedManager(atx)
		w, err := fm.EntityWriter(ctx, fvid)
		if err != nil {
			t.Fatal(err)
		}
		fviresult, err := importFeedVersion(ctx, fm, w, fv, Options{Storage: "/"})
		if err != nil {
			t.Error(err)
		}
		// Check
		count := 0
		expstops := testreader.ExampleZip.Counts["stops.txt"]
		testdb.ShouldGet(t, atx, &count, "SELECT count(*) FROM gtfs_stops WHERE feed_version_id = ?", fvid)
		if count != expstops {
			t.Errorf("expect %d stops, got %d", expstops, count)
		}
		expstopcount := fviresult.EntityCount["stops.txt"]
		if count != expstopcount {
			t.Errorf("expect %d stops in fvi result, got %d", count, expstopcount)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}
