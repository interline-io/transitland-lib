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

type Job = jobs.Job
type Backend = jobs.Backend
type Runner = jobs.Runner
type JobMiddleware = jobs.JobMiddleware
type JobWorker = jobs.JobWorker
type JobArgs = jobs.JobArgs

// TestSetup is the per-test pair returned by the factory. Tests register
// workers/middleware on Runner and submit jobs through Backend; the factory
// is responsible for wiring Backend to Runner internally.
type TestSetup struct {
	Runner  *Runner
	Backend Backend
}

var (
	feeds = []string{"BA", "SF", "AC", "CT"}
)

type TestWorker struct {
	kind  string
	count *int64
}

func (t *TestWorker) Kind() string {
	return t.kind
}

func (t *TestWorker) Run(ctx context.Context) error {
	time.Sleep(1 * time.Millisecond)
	atomic.AddInt64(t.count, 1)
	return nil
}

func checkErr(t testing.TB, err error) {
	if err != nil {
		t.Fatal(err)
	}
}

// adminCtx is the standard context used by tests to call status/watch/list APIs.
// Adapters require an authenticated user for these methods.
func adminCtx() context.Context {
	user := authn.NewCtxUser("test-admin", "", "").WithRoles("admin")
	return authn.WithUser(context.Background(), user)
}

func userCtx(id string) context.Context {
	return authn.WithUser(context.Background(), authn.NewCtxUser(id, "", ""))
}

// TestBackend runs the full conformance suite — enqueue/run behavior plus the
// lifecycle methods (Status, Watch, ListJobs, Cancel) and access control.
// Backends that haven't yet implemented the lifecycle methods should call
// TestBackendCore instead.
func TestBackend(t *testing.T, newSetup func(string) TestSetup) {
	TestBackendCore(t, newSetup)
	TestBackendLifecycle(t, newSetup)
}

// TestBackendCore exercises the original enqueue/run/middleware behavior any
// backend should support.
func TestBackendCore(t *testing.T, newSetup func(string) TestSetup) {
	ctx := adminCtx()
	queueName := func(t testing.TB) string {
		tName := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
		return fmt.Sprintf("%s-%d-%d", tName, os.Getpid(), time.Now().UnixNano())
	}
	sleepyTime := 3 * time.Second
	t.Run("run", func(t *testing.T) {
		setup := newSetup(queueName(t))
		count := int64(0)
		checkErr(t, setup.Runner.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testRun"} }))
		for _, feed := range feeds {
			if err := setup.Runner.Run(ctx, Job{Kind: "testRun", Args: JobArgs{"feed_id": feed}}); err != nil {
				t.Fatal(err)
			}
		}
		assert.Equal(t, len(feeds), int(count))
	})
	t.Run("simple", func(t *testing.T) {
		setup := newSetup(queueName(t))
		count := int64(0)
		checkErr(t, setup.Runner.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "test"} }))
		for _, feed := range feeds {
			if _, err := setup.Backend.AddJob(ctx, Job{Kind: "test", Args: JobArgs{"feed_id": feed}}); err != nil {
				t.Fatal(err)
			}
		}
		go func() {
			time.Sleep(sleepyTime)
			setup.Backend.Stop(ctx)
		}()
		if err := setup.Backend.Run(ctx); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, len(feeds), int(count))
	})
	t.Run("AddJobs", func(t *testing.T) {
		setup := newSetup(queueName(t))
		count := int64(0)
		checkErr(t, setup.Runner.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testAddJobs"} }))
		var jj []Job
		for i := 0; i < 10; i++ {
			jj = append(jj, Job{Kind: "testAddJobs", Args: JobArgs{"test": fmt.Sprintf("n:%d", i)}})
		}
		_, err := setup.Backend.AddJobs(ctx, jj)
		checkErr(t, err)
		go func() {
			time.Sleep(sleepyTime)
			setup.Backend.Stop(ctx)
		}()
		if err := setup.Backend.Run(ctx); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, int64(10), count)
	})
	t.Run("unique", func(t *testing.T) {
		setup := newSetup(queueName(t))
		count := int64(0)
		checkErr(t, setup.Runner.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testUnique"} }))
		checkErr(t, setup.Runner.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testNotUnique"} }))

		for i := 0; i < 10; i++ {
			// 1 job: j=0
			for j := 0; j < 10; j++ {
				job := Job{Kind: "testUnique", Unique: true, Args: JobArgs{"test": fmt.Sprintf("n:%d", j/10)}}
				_, err := setup.Backend.AddJob(ctx, job)
				checkErr(t, err)
			}
			// 3 jobs; j=3, j=6, j=9... j=0 is not unique
			for j := 0; j < 10; j++ {
				job := Job{Kind: "testUnique", Unique: true, Args: JobArgs{"test": fmt.Sprintf("n:%d", j/3)}}
				_, err := setup.Backend.AddJob(ctx, job)
				checkErr(t, err)
			}
			// 10 jobs: j=0, j=0, j=2, j=2, j=4, j=4, j=6 j=6, j=8, j=8
			for j := 0; j < 10; j++ {
				job := Job{Kind: "testNotUnique", Args: JobArgs{"test": fmt.Sprintf("n:%d", j/2)}}
				_, err := setup.Backend.AddJob(ctx, job)
				checkErr(t, err)
			}
		}
		go func() {
			time.Sleep(sleepyTime)
			setup.Backend.Stop(ctx)
		}()
		if err := setup.Backend.Run(ctx); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, int64(104), count)
	})
	t.Run("deadline", func(t *testing.T) {
		setup := newSetup(queueName(t))
		count := int64(0)
		checkErr(t, setup.Runner.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testDeadline"} }))
		setup.Backend.AddJob(ctx, Job{Kind: "testDeadline", Args: JobArgs{"test": "test"}, Deadline: 0})
		setup.Backend.AddJob(ctx, Job{Kind: "testDeadline", Args: JobArgs{"test": "test"}, Deadline: time.Now().Add(1 * time.Hour).Unix()})
		setup.Backend.AddJob(ctx, Job{Kind: "testDeadline", Args: JobArgs{"test": "test"}, Deadline: time.Now().Add(-1 * time.Hour).Unix()})
		go func() {
			time.Sleep(sleepyTime)
			setup.Backend.Stop(ctx)
		}()
		if err := setup.Backend.Run(ctx); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, int64(2), count)
	})
	t.Run("middleware", func(t *testing.T) {
		setup := newSetup(queueName(t))
		jwCount := int64(0)
		setup.Runner.Use(func(w JobWorker, j Job) JobWorker {
			return &testJobMiddleware{
				JobWorker: w,
				jobCount:  &jwCount,
			}
		})
		count := int64(0)
		checkErr(t, setup.Runner.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testMiddleware"} }))
		setup.Backend.AddJob(ctx, Job{Kind: "testMiddleware", Args: JobArgs{"mw": "ok1"}})
		setup.Backend.AddJob(ctx, Job{Kind: "testMiddleware", Args: JobArgs{"mw": "ok2"}})
		go func() {
			time.Sleep(sleepyTime)
			setup.Backend.Stop(ctx)
		}()
		if err := setup.Backend.Run(ctx); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, int64(2), count)
		assert.Equal(t, int64(2*10), jwCount)
	})
}

// requireReporter asserts the backend implements JobStatusReporter and skips
// the sub-test if it doesn't. Lifecycle backends that don't track jobs
// (e.g. Redis fire-and-forget) won't satisfy this.
func requireReporter(t *testing.T, b Backend) jobs.JobStatusReporter {
	t.Helper()
	r, ok := b.(jobs.JobStatusReporter)
	if !ok {
		t.Skipf("backend %T does not implement jobs.JobStatusReporter", b)
	}
	return r
}

// TestBackendLifecycle exercises Status, Watch, ListJobs, Cancel and the
// admin-or-creator access rules. Sub-tests skip when the backend isn't a
// JobStatusReporter.
func TestBackendLifecycle(t *testing.T, newSetup func(string) TestSetup) {
	ctx := adminCtx()
	queueName := func(t testing.TB) string {
		tName := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
		return fmt.Sprintf("%s-%d-%d", tName, os.Getpid(), time.Now().UnixNano())
	}
	sleepyTime := 3 * time.Second
	t.Run("status", func(t *testing.T) {
		setup := newSetup(queueName(t))
		reporter := requireReporter(t, setup.Backend)
		count := int64(0)
		checkErr(t, setup.Runner.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testStatus"} }))
		st, err := setup.Backend.AddJob(ctx, Job{Kind: "testStatus", UserID: "alice", Args: JobArgs{"x": "1"}})
		checkErr(t, err)
		assert.NotEmpty(t, st.Job.ID, "AddJob should assign a JobId")
		assert.Equal(t, "alice", st.Job.UserID)
		queuedSt, err := reporter.Status(ctx, st.Job.ID)
		checkErr(t, err)
		assert.Equal(t, st.Job.ID, queuedSt.Job.ID)
		go func() {
			time.Sleep(sleepyTime)
			setup.Backend.Stop(ctx)
		}()
		if err := setup.Backend.Run(ctx); err != nil {
			t.Fatal(err)
		}
		got, err := reporter.Status(ctx, st.Job.ID)
		checkErr(t, err)
		assert.True(t, got.State.Terminal(), "expected terminal state, got %s", got.State)
		assert.Equal(t, jobs.JobStateSucceeded, got.State)
		_, err = reporter.Status(ctx, "does-not-exist")
		assert.Error(t, err)
	})
	t.Run("watch", func(t *testing.T) {
		setup := newSetup(queueName(t))
		reporter := requireReporter(t, setup.Backend)
		count := int64(0)
		checkErr(t, setup.Runner.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testWatch"} }))
		st, err := setup.Backend.AddJob(ctx, Job{Kind: "testWatch", UserID: "alice", Args: JobArgs{"x": "live"}})
		checkErr(t, err)
		// Open the watch before the queue starts so we observe the full transition.
		ch, err := reporter.Watch(ctx, st.Job.ID)
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
		setup := newSetup(queueName(t))
		reporter := requireReporter(t, setup.Backend)
		// Per-run-unique JobType so persistent backends don't see rows from prior runs.
		listType := fmt.Sprintf("testList-%d-%d", os.Getpid(), time.Now().UnixNano())
		count := int64(0)
		checkErr(t, setup.Runner.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: listType} }))
		for i := 0; i < 5; i++ {
			_, err := setup.Backend.AddJob(ctx, Job{Kind: listType, UserID: "alice", Args: JobArgs{"i": i}})
			checkErr(t, err)
			time.Sleep(1 * time.Millisecond)
		}
		for i := 0; i < 3; i++ {
			_, err := setup.Backend.AddJob(ctx, Job{Kind: listType, UserID: "bob", Args: JobArgs{"i": i}})
			checkErr(t, err)
			time.Sleep(1 * time.Millisecond)
		}
		all, err := reporter.ListJobs(ctx, jobs.JobListOptions{Kind: listType})
		checkErr(t, err)
		assert.Equal(t, 8, len(all.Jobs))
		aliceCtx := userCtx("alice")
		mine, err := reporter.ListJobs(aliceCtx, jobs.JobListOptions{Kind: listType, UserID: "bob"})
		checkErr(t, err)
		assert.Equal(t, 5, len(mine.Jobs))
		for _, st := range mine.Jobs {
			assert.Equal(t, "alice", st.Job.UserID)
		}
		seen := map[string]bool{}
		var cursor string
		for pages := 0; pages < 10; pages++ {
			page, err := reporter.ListJobs(ctx, jobs.JobListOptions{Kind: listType, UserID: "alice", Limit: 2, After: cursor})
			checkErr(t, err)
			for _, st := range page.Jobs {
				assert.False(t, seen[st.Job.ID], "duplicate JobId across pages")
				seen[st.Job.ID] = true
			}
			if page.NextCursor == "" {
				break
			}
			cursor = page.NextCursor
		}
		assert.Equal(t, 5, len(seen))
	})
	t.Run("auth", func(t *testing.T) {
		setup := newSetup(queueName(t))
		reporter := requireReporter(t, setup.Backend)
		count := int64(0)
		checkErr(t, setup.Runner.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testAuth"} }))
		st, err := setup.Backend.AddJob(ctx, Job{Kind: "testAuth", UserID: "alice", Args: JobArgs{"x": "1"}})
		checkErr(t, err)
		if _, err := reporter.Status(userCtx("alice"), st.Job.ID); err != nil {
			t.Errorf("owner Status: %v", err)
		}
		if _, err := reporter.Status(ctx, st.Job.ID); err != nil {
			t.Errorf("admin Status: %v", err)
		}
		_, err = reporter.Status(userCtx("bob"), st.Job.ID)
		assert.ErrorIs(t, err, jobs.ErrJobAccessDenied)
		_, err = reporter.Watch(userCtx("bob"), st.Job.ID)
		assert.ErrorIs(t, err, jobs.ErrJobAccessDenied)
		_, err = reporter.ListJobs(context.Background(), jobs.JobListOptions{})
		assert.ErrorIs(t, err, jobs.ErrJobAccessDenied)
	})
	t.Run("cancel", func(t *testing.T) {
		setup := newSetup(queueName(t))
		reporter := requireReporter(t, setup.Backend)
		count := int64(0)
		checkErr(t, setup.Runner.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testCancel"} }))
		// Submit, cancel before Run drains the queue, confirm terminal state is cancelled and worker never ran.
		st, err := setup.Backend.AddJob(ctx, Job{Kind: "testCancel", UserID: "alice", Args: JobArgs{"x": "1"}})
		checkErr(t, err)
		checkErr(t, reporter.Cancel(ctx, st.Job.ID))
		// Stranger can't cancel.
		err = reporter.Cancel(userCtx("bob"), st.Job.ID)
		assert.ErrorIs(t, err, jobs.ErrJobAccessDenied)
		// Cancel of unknown job.
		err = reporter.Cancel(ctx, "does-not-exist")
		assert.Error(t, err)
		// Run the queue and confirm the cancelled job didn't execute.
		go func() {
			time.Sleep(sleepyTime)
			setup.Backend.Stop(ctx)
		}()
		if err := setup.Backend.Run(ctx); err != nil {
			t.Fatal(err)
		}
		final, err := reporter.Status(ctx, st.Job.ID)
		checkErr(t, err)
		assert.Equal(t, jobs.JobStateCancelled, final.State)
		assert.Equal(t, int64(0), atomic.LoadInt64(&count), "cancelled job should not have run")
	})
}

type testJobMiddleware struct {
	jobCount *int64
	JobWorker
}

func (w *testJobMiddleware) Run(ctx context.Context) error {
	atomic.AddInt64(w.jobCount, 10)
	if err := w.JobWorker.Run(ctx); err != nil {
		return err
	}
	return nil
}
