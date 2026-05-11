package jobtest

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/jobs"
	"github.com/stretchr/testify/assert"
)

// TestSetup is the per-test triple returned by the factory. QueueName is the
// name to pass to Backend.Queue (persistent backends namespace by it).
type TestSetup struct {
	Runner    *jobs.Runner
	Backend   jobs.Backend
	QueueName string
}

// Queue is a test convenience that panics on error; the test factory must
// have registered QueueName with the backend.
func (s TestSetup) Queue() jobs.Queue {
	q, err := s.Backend.Queue(s.QueueName)
	if err != nil {
		panic(err)
	}
	return q
}

var feeds = []string{"BA", "SF", "AC", "CT"}

type testWorker struct {
	kind  string
	count *int64
}

func (t *testWorker) Kind() string { return t.kind }
func (t *testWorker) Run(ctx context.Context) error {
	time.Sleep(1 * time.Millisecond)
	atomic.AddInt64(t.count, 1)
	return nil
}

func checkErr(t testing.TB, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

// adminCtx provides an authenticated admin user; required by Status/Watch/List.
func adminCtx() context.Context {
	user := authn.NewCtxUser("test-admin", "", "").WithRoles("admin")
	return authn.WithUser(context.Background(), user)
}

func userCtx(id string) context.Context {
	return authn.WithUser(context.Background(), authn.NewCtxUser(id, "", ""))
}

// runUntil schedules Stop after d, then runs Backend.Run synchronously.
func runUntil(t testing.TB, setup TestSetup, ctx context.Context, d time.Duration) {
	t.Helper()
	go func() {
		time.Sleep(d)
		setup.Backend.Stop(ctx)
	}()
	if err := setup.Backend.Run(ctx); err != nil {
		t.Fatal(err)
	}
}

// startBackend runs Backend.Run in a goroutine; defer the returned stop().
func startBackend(setup TestSetup) (stop func()) {
	runCtx, cancel := context.WithCancel(context.Background())
	go func() { _ = setup.Backend.Run(runCtx) }()
	return func() {
		_ = setup.Backend.Stop(runCtx)
		cancel()
	}
}

func uniqueQueueName(t testing.TB) string {
	tName := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
	// River caps queue names at 64 chars; cap the test-name portion so an
	// adapter prefix + pid + nanos suffix still fits.
	if len(tName) > 28 {
		tName = tName[:28]
	}
	return fmt.Sprintf("%s-%d-%d", tName, os.Getpid(), time.Now().UnixNano())
}

// TestBackend runs the full conformance suite (Core + Lifecycle).
// Backends without StatusQueue support should call TestBackendCore instead.
func TestBackend(t *testing.T, newSetup func(string) TestSetup) {
	TestBackendCore(t, newSetup)
	TestBackendLifecycle(t, newSetup)
}

func TestBackendCore(t *testing.T, newSetup func(string) TestSetup) {
	ctx := adminCtx()
	sleepyTime := 3 * time.Second
	t.Run("run", func(t *testing.T) {
		setup := newSetup(uniqueQueueName(t))
		count := int64(0)
		checkErr(t, setup.Runner.Register(func() jobs.Worker { return &testWorker{count: &count, kind: "testRun"} }))
		for _, feed := range feeds {
			if err := setup.Runner.Run(ctx, jobs.Job{Kind: "testRun", Args: jobs.Args{"feed_id": feed}}); err != nil {
				t.Fatal(err)
			}
		}
		assert.Equal(t, len(feeds), int(count))
	})
	t.Run("simple", func(t *testing.T) {
		setup := newSetup(uniqueQueueName(t))
		count := int64(0)
		checkErr(t, setup.Runner.Register(func() jobs.Worker { return &testWorker{count: &count, kind: "test"} }))
		q := setup.Queue()
		for _, feed := range feeds {
			if _, err := q.Submit(ctx, jobs.Job{Kind: "test", Args: jobs.Args{"feed_id": feed}}); err != nil {
				t.Fatal(err)
			}
		}
		runUntil(t, setup, ctx, sleepyTime)
		assert.Equal(t, len(feeds), int(count))
	})
	t.Run("SubmitMany", func(t *testing.T) {
		setup := newSetup(uniqueQueueName(t))
		count := int64(0)
		checkErr(t, setup.Runner.Register(func() jobs.Worker { return &testWorker{count: &count, kind: "testSubmitMany"} }))
		var jj []jobs.Job
		for i := 0; i < 10; i++ {
			jj = append(jj, jobs.Job{Kind: "testSubmitMany", Args: jobs.Args{"test": fmt.Sprintf("n:%d", i)}})
		}
		_, err := setup.Queue().SubmitMany(ctx, jj)
		checkErr(t, err)
		runUntil(t, setup, ctx, sleepyTime)
		assert.Equal(t, int64(10), count)
	})
	t.Run("unique", func(t *testing.T) {
		setup := newSetup(uniqueQueueName(t))
		count := int64(0)
		checkErr(t, setup.Runner.Register(func() jobs.Worker { return &testWorker{count: &count, kind: "testUnique"} }))
		checkErr(t, setup.Runner.Register(func() jobs.Worker { return &testWorker{count: &count, kind: "testNotUnique"} }))
		q := setup.Queue()
		for i := 0; i < 10; i++ {
			// 1 job: j=0
			for j := 0; j < 10; j++ {
				job := jobs.Job{Kind: "testUnique", Opts: jobs.JobOpts{Unique: true}, Args: jobs.Args{"test": fmt.Sprintf("n:%d", j/10)}}
				_, err := q.Submit(ctx, job)
				checkErr(t, err)
			}
			// 3 jobs; j=3, j=6, j=9
			for j := 0; j < 10; j++ {
				job := jobs.Job{Kind: "testUnique", Opts: jobs.JobOpts{Unique: true}, Args: jobs.Args{"test": fmt.Sprintf("n:%d", j/3)}}
				_, err := q.Submit(ctx, job)
				checkErr(t, err)
			}
			// 10 jobs (not unique)
			for j := 0; j < 10; j++ {
				job := jobs.Job{Kind: "testNotUnique", Args: jobs.Args{"test": fmt.Sprintf("n:%d", j/2)}}
				_, err := q.Submit(ctx, job)
				checkErr(t, err)
			}
		}
		runUntil(t, setup, ctx, sleepyTime)
		assert.Equal(t, int64(104), count)
	})
	t.Run("deadline", func(t *testing.T) {
		setup := newSetup(uniqueQueueName(t))
		count := int64(0)
		checkErr(t, setup.Runner.Register(func() jobs.Worker { return &testWorker{count: &count, kind: "testDeadline"} }))
		q := setup.Queue()
		q.Submit(ctx, jobs.Job{Kind: "testDeadline", Args: jobs.Args{"test": "test"}})
		q.Submit(ctx, jobs.Job{Kind: "testDeadline", Args: jobs.Args{"test": "test"}, Opts: jobs.JobOpts{Deadline: time.Now().Add(1 * time.Hour)}})
		q.Submit(ctx, jobs.Job{Kind: "testDeadline", Args: jobs.Args{"test": "test"}, Opts: jobs.JobOpts{Deadline: time.Now().Add(-1 * time.Hour)}})
		runUntil(t, setup, ctx, sleepyTime)
		assert.Equal(t, int64(2), count)
	})
	t.Run("middleware", func(t *testing.T) {
		setup := newSetup(uniqueQueueName(t))
		jwCount := int64(0)
		setup.Runner.Use(func(w jobs.Worker, j jobs.Job) jobs.Worker {
			return &testMiddleware{Worker: w, count: &jwCount}
		})
		count := int64(0)
		checkErr(t, setup.Runner.Register(func() jobs.Worker { return &testWorker{count: &count, kind: "testMiddleware"} }))
		q := setup.Queue()
		q.Submit(ctx, jobs.Job{Kind: "testMiddleware", Args: jobs.Args{"mw": "ok1"}})
		q.Submit(ctx, jobs.Job{Kind: "testMiddleware", Args: jobs.Args{"mw": "ok2"}})
		runUntil(t, setup, ctx, sleepyTime)
		assert.Equal(t, int64(2), count)
		assert.Equal(t, int64(2*10), jwCount)
	})
	t.Run("middlewareOrder", func(t *testing.T) {
		// Use(A); Use(B); Use(C) → A enters first, exits last.
		setup := newSetup(uniqueQueueName(t))
		var entry, exit []string
		mk := func(label string) jobs.Middleware {
			return func(w jobs.Worker, j jobs.Job) jobs.Worker {
				return &recordingMiddleware{Worker: w, label: label, entry: &entry, exit: &exit}
			}
		}
		setup.Runner.Use(mk("A"))
		setup.Runner.Use(mk("B"))
		setup.Runner.Use(mk("C"))
		count := int64(0)
		checkErr(t, setup.Runner.Register(func() jobs.Worker { return &testWorker{count: &count, kind: "testMwOrder"} }))
		q := setup.Queue()
		q.Submit(ctx, jobs.Job{Kind: "testMwOrder"})
		runUntil(t, setup, ctx, sleepyTime)
		assert.Equal(t, []string{"A", "B", "C"}, entry, "entry order should match registration order")
		assert.Equal(t, []string{"C", "B", "A"}, exit, "exit order should be reverse of entry")
	})
}

// requireStatusQueue skips the sub-test if the queue isn't a StatusQueue
// (e.g. fire-and-forget Redis).
func requireStatusQueue(t *testing.T, q jobs.Queue) jobs.StatusQueue {
	t.Helper()
	sq, ok := q.(jobs.StatusQueue)
	if !ok {
		t.Skipf("queue %T does not implement jobs.StatusQueue", q)
	}
	return sq
}

// TestBackendLifecycle exercises StatusQueue methods and access rules.
// Sub-tests skip when the queue isn't a StatusQueue.
func TestBackendLifecycle(t *testing.T, newSetup func(string) TestSetup) {
	ctx := adminCtx()
	sleepyTime := 3 * time.Second
	t.Run("status", func(t *testing.T) {
		setup := newSetup(uniqueQueueName(t))
		sq := requireStatusQueue(t, setup.Queue())
		count := int64(0)
		checkErr(t, setup.Runner.Register(func() jobs.Worker { return &testWorker{count: &count, kind: "testStatus"} }))
		st, err := setup.Queue().Submit(ctx, jobs.Job{Kind: "testStatus", Args: jobs.Args{"x": "1"}, Opts: jobs.JobOpts{UserID: "alice"}})
		checkErr(t, err)
		assert.NotEmpty(t, st.Job.ID, "Submit should assign a Job ID")
		assert.Equal(t, "alice", st.Job.Opts.UserID)
		queuedSt, err := sq.Status(ctx, st.Job.ID)
		checkErr(t, err)
		assert.Equal(t, st.Job.ID, queuedSt.Job.ID)
		runUntil(t, setup, ctx, sleepyTime)
		got, err := sq.Status(ctx, st.Job.ID)
		checkErr(t, err)
		assert.True(t, got.State.Terminal(), "expected terminal state, got %s", got.State)
		assert.Equal(t, jobs.JobStateSucceeded, got.State)
		_, err = sq.Status(ctx, "does-not-exist")
		assert.Error(t, err)
	})
	t.Run("watch", func(t *testing.T) {
		setup := newSetup(uniqueQueueName(t))
		sq := requireStatusQueue(t, setup.Queue())
		count := int64(0)
		checkErr(t, setup.Runner.Register(func() jobs.Worker { return &testWorker{count: &count, kind: "testWatch"} }))
		st, err := setup.Queue().Submit(ctx, jobs.Job{Kind: "testWatch", Args: jobs.Args{"x": "live"}, Opts: jobs.JobOpts{UserID: "alice"}})
		checkErr(t, err)
		ch, err := sq.Watch(ctx, st.Job.ID)
		checkErr(t, err)
		go func() {
			time.Sleep(sleepyTime)
			setup.Backend.Stop(ctx)
		}()
		go func() { _ = setup.Backend.Run(ctx) }()
		var events []jobs.JobEvent
		drained := make(chan struct{})
		go func() {
			for ev := range ch {
				events = append(events, ev)
			}
			close(drained)
		}()
		select {
		case <-drained:
		case <-time.After(sleepyTime + 2*time.Second):
			t.Fatal("watch did not close")
		}
		if assert.NotEmpty(t, events, "expected at least one terminal event") {
			assert.True(t, events[len(events)-1].State.Terminal())
		}
	})
	t.Run("list", func(t *testing.T) {
		setup := newSetup(uniqueQueueName(t))
		sq := requireStatusQueue(t, setup.Queue())
		// Per-run-unique Kind so persistent backends don't see prior rows.
		listKind := fmt.Sprintf("testList-%d-%d", os.Getpid(), time.Now().UnixNano())
		count := int64(0)
		checkErr(t, setup.Runner.Register(func() jobs.Worker { return &testWorker{count: &count, kind: listKind} }))
		q := setup.Queue()
		for i := 0; i < 5; i++ {
			_, err := q.Submit(ctx, jobs.Job{Kind: listKind, Args: jobs.Args{"i": i}, Opts: jobs.JobOpts{UserID: "alice"}})
			checkErr(t, err)
			time.Sleep(1 * time.Millisecond)
		}
		for i := 0; i < 3; i++ {
			_, err := q.Submit(ctx, jobs.Job{Kind: listKind, Args: jobs.Args{"i": i}, Opts: jobs.JobOpts{UserID: "bob"}})
			checkErr(t, err)
			time.Sleep(1 * time.Millisecond)
		}
		all, err := sq.List(ctx, jobs.ListOptions{Kind: listKind})
		checkErr(t, err)
		assert.Equal(t, 8, len(all.Jobs))
		aliceCtx := userCtx("alice")
		mine, err := sq.List(aliceCtx, jobs.ListOptions{Kind: listKind, UserID: "bob"})
		checkErr(t, err)
		assert.Equal(t, 5, len(mine.Jobs))
		for _, st := range mine.Jobs {
			assert.Equal(t, "alice", st.Job.Opts.UserID)
		}
		seen := map[string]bool{}
		offset := 0
		for pages := 0; pages < 10; pages++ {
			page, err := sq.List(ctx, jobs.ListOptions{Kind: listKind, UserID: "alice", Limit: 2, Offset: offset})
			checkErr(t, err)
			if len(page.Jobs) == 0 {
				break
			}
			for _, st := range page.Jobs {
				assert.False(t, seen[st.Job.ID], "duplicate Job ID across pages")
				seen[st.Job.ID] = true
			}
			offset += len(page.Jobs)
		}
		assert.Equal(t, 5, len(seen))
	})
	t.Run("auth", func(t *testing.T) {
		setup := newSetup(uniqueQueueName(t))
		sq := requireStatusQueue(t, setup.Queue())
		count := int64(0)
		checkErr(t, setup.Runner.Register(func() jobs.Worker { return &testWorker{count: &count, kind: "testAuth"} }))
		st, err := setup.Queue().Submit(ctx, jobs.Job{Kind: "testAuth", Args: jobs.Args{"x": "1"}, Opts: jobs.JobOpts{UserID: "alice"}})
		checkErr(t, err)
		if _, err := sq.Status(userCtx("alice"), st.Job.ID); err != nil {
			t.Errorf("owner Status: %v", err)
		}
		if _, err := sq.Status(ctx, st.Job.ID); err != nil {
			t.Errorf("admin Status: %v", err)
		}
		_, err = sq.Status(userCtx("bob"), st.Job.ID)
		assert.ErrorIs(t, err, jobs.ErrJobAccessDenied)
		_, err = sq.Watch(userCtx("bob"), st.Job.ID)
		assert.ErrorIs(t, err, jobs.ErrJobAccessDenied)
		_, err = sq.List(context.Background(), jobs.ListOptions{})
		assert.ErrorIs(t, err, jobs.ErrJobAccessDenied)
	})
	t.Run("cancel", func(t *testing.T) {
		setup := newSetup(uniqueQueueName(t))
		sq := requireStatusQueue(t, setup.Queue())
		count := int64(0)
		checkErr(t, setup.Runner.Register(func() jobs.Worker { return &testWorker{count: &count, kind: "testCancel"} }))
		st, err := setup.Queue().Submit(ctx, jobs.Job{Kind: "testCancel", Args: jobs.Args{"x": "1"}, Opts: jobs.JobOpts{UserID: "alice"}})
		checkErr(t, err)
		checkErr(t, sq.Cancel(ctx, st.Job.ID))
		err = sq.Cancel(userCtx("bob"), st.Job.ID)
		assert.ErrorIs(t, err, jobs.ErrJobAccessDenied)
		err = sq.Cancel(ctx, "does-not-exist")
		assert.Error(t, err)
		runUntil(t, setup, ctx, sleepyTime)
		final, err := sq.Status(ctx, st.Job.ID)
		checkErr(t, err)
		assert.Equal(t, jobs.JobStateCancelled, final.State)
		assert.Equal(t, int64(0), atomic.LoadInt64(&count), "cancelled job should not have run")
	})
}

type testMiddleware struct {
	count *int64
	jobs.Worker
}

func (w *testMiddleware) Run(ctx context.Context) error {
	atomic.AddInt64(w.count, 10)
	return w.Worker.Run(ctx)
}

// recordingMiddleware appends label to entry on the way in and exit on the
// way out. Used to assert middleware execution order. Single-job tests only;
// no synchronization.
type recordingMiddleware struct {
	jobs.Worker
	label string
	entry *[]string
	exit  *[]string
}

func (w *recordingMiddleware) Run(ctx context.Context) error {
	*w.entry = append(*w.entry, w.label)
	err := w.Worker.Run(ctx)
	*w.exit = append(*w.exit, w.label)
	return err
}
