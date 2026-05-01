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
type JobQueue = jobs.JobQueue
type JobMiddleware = jobs.JobMiddleware
type JobWorker = jobs.JobWorker
type JobArgs = jobs.JobArgs

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

// TestJobQueue runs the full conformance suite — legacy enqueue/run behavior
// plus the lifecycle methods (Status, Watch, ListJobs) and access control.
// Backends that haven't yet implemented the lifecycle methods should call
// TestJobQueueCore instead.
func TestJobQueue(t *testing.T, newQueue func(string) JobQueue) {
	TestJobQueueCore(t, newQueue)
	TestJobQueueLifecycle(t, newQueue)
}

// TestJobQueueCore exercises the original enqueue/run/middleware behavior
// any backend should support.
func TestJobQueueCore(t *testing.T, newQueue func(string) JobQueue) {
	ctx := adminCtx()
	queueName := func(t testing.TB) string {
		tName := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
		return fmt.Sprintf("%s-%d-%d", tName, os.Getpid(), time.Now().UnixNano())
	}
	sleepyTime := 3 * time.Second
	t.Run("run", func(t *testing.T) {
		rtJobs := newQueue(queueName(t))
		count := int64(0)
		checkErr(t, rtJobs.AddJobType(func() JobWorker { return &TestWorker{count: &count, kind: "testRun"} }))
		for _, feed := range feeds {
			if _, err := rtJobs.RunJob(ctx, Job{JobType: "testRun", JobArgs: JobArgs{"feed_id": feed}}); err != nil {
				t.Fatal(err)
			}
		}
		assert.Equal(t, len(feeds), int(count))
	})
	t.Run("simple", func(t *testing.T) {
		// Ugly :(
		rtJobs := newQueue(queueName(t))
		// Add workers
		count := int64(0)
		checkErr(t, rtJobs.AddJobType(func() JobWorker { return &TestWorker{count: &count, kind: "test"} }))

		// Add jobs
		for _, feed := range feeds {
			if _, err := rtJobs.AddJob(ctx, Job{JobType: "test", JobArgs: JobArgs{"feed_id": feed}}); err != nil {
				t.Fatal(err)
			}
		}
		// Run
		go func() {
			time.Sleep(sleepyTime)
			rtJobs.Stop(ctx)
		}()
		if err := rtJobs.Run(ctx); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, len(feeds), int(count))
	})
	t.Run("AddJobs", func(t *testing.T) {
		// Abuse the job queue
		rtJobs := newQueue(queueName(t))
		// Add workers
		count := int64(0)
		checkErr(t, rtJobs.AddJobType(func() JobWorker { return &TestWorker{count: &count, kind: "testAddJobs"} }))
		// Add jobs
		var jobs []Job
		for i := 0; i < 10; i++ {
			// 1 job: j=0
			jobs = append(jobs, Job{JobType: "testAddJobs", JobArgs: JobArgs{"test": fmt.Sprintf("n:%d", i)}})
		}
		// Run
		_, err := rtJobs.AddJobs(ctx, jobs)
		checkErr(t, err)
		go func() {
			time.Sleep(sleepyTime)
			rtJobs.Stop(ctx)
		}()
		if err := rtJobs.Run(ctx); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, int64(10), count)
	})
	t.Run("unique", func(t *testing.T) {
		// Abuse the job queue
		rtJobs := newQueue(queueName(t))
		// Add workers
		count := int64(0)
		checkErr(t, rtJobs.AddJobType(func() JobWorker { return &TestWorker{count: &count, kind: "testUnique"} }))
		checkErr(t, rtJobs.AddJobType(func() JobWorker { return &TestWorker{count: &count, kind: "testNotUnique"} }))

		// Add jobs
		for i := 0; i < 10; i++ {
			// 1 job: j=0
			for j := 0; j < 10; j++ {
				job := Job{JobType: "testUnique", Unique: true, JobArgs: JobArgs{"test": fmt.Sprintf("n:%d", j/10)}}
				_, err := rtJobs.AddJob(ctx, job)
				checkErr(t, err)
			}
			// 3 jobs; j=3, j=6, j=9... j=0 is not unique
			for j := 0; j < 10; j++ {
				job := Job{JobType: "testUnique", Unique: true, JobArgs: JobArgs{"test": fmt.Sprintf("n:%d", j/3)}}
				_, err := rtJobs.AddJob(ctx, job)
				checkErr(t, err)
			}
			// 10 jobs: j=0, j=0, j=2, j=2, j=4, j=4, j=6 j=6, j=8, j=8
			for j := 0; j < 10; j++ {
				job := Job{JobType: "testNotUnique", JobArgs: JobArgs{"test": fmt.Sprintf("n:%d", j/2)}}
				_, err := rtJobs.AddJob(ctx, job)
				checkErr(t, err)
			}
		}
		// Run
		go func() {
			time.Sleep(sleepyTime)
			rtJobs.Stop(ctx)
		}()
		if err := rtJobs.Run(ctx); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, int64(104), count)
	})
	t.Run("deadline", func(t *testing.T) {
		rtJobs := newQueue(queueName(t))
		// Add workers
		count := int64(0)
		checkErr(t, rtJobs.AddJobType(func() JobWorker { return &TestWorker{count: &count, kind: "testDeadline"} }))
		// Add jobs
		rtJobs.AddJob(ctx, Job{JobType: "testDeadline", JobArgs: JobArgs{"test": "test"}, JobDeadline: 0})
		rtJobs.AddJob(ctx, Job{JobType: "testDeadline", JobArgs: JobArgs{"test": "test"}, JobDeadline: time.Now().Add(1 * time.Hour).Unix()})
		rtJobs.AddJob(ctx, Job{JobType: "testDeadline", JobArgs: JobArgs{"test": "test"}, JobDeadline: time.Now().Add(-1 * time.Hour).Unix()})
		// Run
		go func() {
			time.Sleep(sleepyTime)
			rtJobs.Stop(ctx)
		}()
		if err := rtJobs.Run(ctx); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, int64(2), count)
	})
	t.Run("middleware", func(t *testing.T) {
		rtJobs := newQueue(queueName(t))
		// Add middleware
		jwCount := int64(0)
		rtJobs.Use(func(w JobWorker, j Job) JobWorker {
			return &testJobMiddleware{
				JobWorker: w,
				jobCount:  &jwCount,
			}
		})
		// Add workers
		count := int64(0)
		checkErr(t, rtJobs.AddJobType(func() JobWorker { return &TestWorker{count: &count, kind: "testMiddleware"} }))
		rtJobs.AddJob(ctx, Job{JobType: "testMiddleware", JobArgs: JobArgs{"mw": "ok1"}})
		rtJobs.AddJob(ctx, Job{JobType: "testMiddleware", JobArgs: JobArgs{"mw": "ok2"}})
		// Run
		go func() {
			time.Sleep(sleepyTime)
			rtJobs.Stop(ctx)
		}()
		if err := rtJobs.Run(ctx); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, int64(2), count)
		assert.Equal(t, int64(2*10), jwCount)
	})
}

// TestJobQueueLifecycle exercises Status, Watch, ListJobs, and the
// admin-or-creator access rules. Backends that haven't implemented these
// methods should skip this suite.
func TestJobQueueLifecycle(t *testing.T, newQueue func(string) JobQueue) {
	ctx := adminCtx()
	queueName := func(t testing.TB) string {
		tName := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
		return fmt.Sprintf("%s-%d-%d", tName, os.Getpid(), time.Now().UnixNano())
	}
	sleepyTime := 3 * time.Second
	t.Run("status", func(t *testing.T) {
		rtJobs := newQueue(queueName(t))
		count := int64(0)
		checkErr(t, rtJobs.AddJobType(func() JobWorker { return &TestWorker{count: &count, kind: "testStatus"} }))
		// AddJob returns queued status with adapter-assigned JobId.
		st, err := rtJobs.AddJob(ctx, Job{JobType: "testStatus", UserId: "alice", JobArgs: JobArgs{"x": "1"}})
		checkErr(t, err)
		assert.NotEmpty(t, st.JobId, "AddJob should assign a JobId")
		assert.Equal(t, "alice", st.UserId)
		// Status before Run is OK — backend should track queued state.
		queuedSt, err := rtJobs.Status(ctx, st.JobId)
		checkErr(t, err)
		assert.Equal(t, st.JobId, queuedSt.JobId)
		// Run the queue and wait for the job to become terminal.
		go func() {
			time.Sleep(sleepyTime)
			rtJobs.Stop(ctx)
		}()
		if err := rtJobs.Run(ctx); err != nil {
			t.Fatal(err)
		}
		got, err := rtJobs.Status(ctx, st.JobId)
		checkErr(t, err)
		assert.True(t, got.State.Terminal(), "expected terminal state, got %s", got.State)
		assert.Equal(t, jobs.JobStateSucceeded, got.State)
		// Unknown id.
		_, err = rtJobs.Status(ctx, "does-not-exist")
		assert.Error(t, err)
	})
	t.Run("watch", func(t *testing.T) {
		rtJobs := newQueue(queueName(t))
		count := int64(0)
		checkErr(t, rtJobs.AddJobType(func() JobWorker { return &TestWorker{count: &count, kind: "testWatch"} }))
		st, err := rtJobs.AddJob(ctx, Job{JobType: "testWatch", UserId: "alice", JobArgs: JobArgs{"x": "live"}})
		checkErr(t, err)
		// Open the watch before the queue starts; we should see the transition
		// from queued through to terminal.
		ch, err := rtJobs.Watch(ctx, st.JobId)
		checkErr(t, err)
		go func() {
			time.Sleep(sleepyTime)
			rtJobs.Stop(ctx)
		}()
		go func() { _ = rtJobs.Run(ctx) }()
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
		rtJobs := newQueue(queueName(t))
		// Use a per-run unique JobType so persistent backends don't see
		// rows from prior runs.
		listType := fmt.Sprintf("testList-%d-%d", os.Getpid(), time.Now().UnixNano())
		count := int64(0)
		checkErr(t, rtJobs.AddJobType(func() JobWorker { return &TestWorker{count: &count, kind: listType} }))
		// Submit 5 jobs as alice, 3 as bob.
		for i := 0; i < 5; i++ {
			_, err := rtJobs.AddJob(ctx, Job{JobType: listType, UserId: "alice", JobArgs: JobArgs{"i": i}})
			checkErr(t, err)
			time.Sleep(1 * time.Millisecond)
		}
		for i := 0; i < 3; i++ {
			_, err := rtJobs.AddJob(ctx, Job{JobType: listType, UserId: "bob", JobArgs: JobArgs{"i": i}})
			checkErr(t, err)
			time.Sleep(1 * time.Millisecond)
		}
		// Admin sees all.
		all, err := rtJobs.ListJobs(ctx, jobs.JobListOptions{JobType: listType})
		checkErr(t, err)
		assert.Equal(t, 8, len(all.Jobs))
		// Alice (non-admin) sees only her own regardless of opts.UserId.
		aliceCtx := userCtx("alice")
		mine, err := rtJobs.ListJobs(aliceCtx, jobs.JobListOptions{JobType: listType, UserId: "bob"})
		checkErr(t, err)
		assert.Equal(t, 5, len(mine.Jobs))
		for _, st := range mine.Jobs {
			assert.Equal(t, "alice", st.UserId)
		}
		// Pagination: admin filters to alice with limit 2, walks pages.
		seen := map[string]bool{}
		var cursor string
		for pages := 0; pages < 10; pages++ {
			page, err := rtJobs.ListJobs(ctx, jobs.JobListOptions{JobType: listType, UserId: "alice", Limit: 2, After: cursor})
			checkErr(t, err)
			for _, st := range page.Jobs {
				assert.False(t, seen[st.JobId], "duplicate JobId across pages")
				seen[st.JobId] = true
			}
			if page.NextCursor == "" {
				break
			}
			cursor = page.NextCursor
		}
		assert.Equal(t, 5, len(seen))
	})
	t.Run("auth", func(t *testing.T) {
		rtJobs := newQueue(queueName(t))
		count := int64(0)
		checkErr(t, rtJobs.AddJobType(func() JobWorker { return &TestWorker{count: &count, kind: "testAuth"} }))
		st, err := rtJobs.AddJob(ctx, Job{JobType: "testAuth", UserId: "alice", JobArgs: JobArgs{"x": "1"}})
		checkErr(t, err)
		// Owner can read.
		if _, err := rtJobs.Status(userCtx("alice"), st.JobId); err != nil {
			t.Errorf("owner Status: %v", err)
		}
		// Admin can read.
		if _, err := rtJobs.Status(ctx, st.JobId); err != nil {
			t.Errorf("admin Status: %v", err)
		}
		// Stranger is denied.
		_, err = rtJobs.Status(userCtx("bob"), st.JobId)
		assert.ErrorIs(t, err, jobs.ErrJobAccessDenied)
		// Watch is denied for stranger.
		_, err = rtJobs.Watch(userCtx("bob"), st.JobId)
		assert.ErrorIs(t, err, jobs.ErrJobAccessDenied)
		// ListJobs without an authenticated user is denied.
		_, err = rtJobs.ListJobs(context.Background(), jobs.JobListOptions{})
		assert.ErrorIs(t, err, jobs.ErrJobAccessDenied)
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
