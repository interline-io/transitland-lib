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
		checkErr(t, rtJobs.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testRun"} }))
		for _, feed := range feeds {
			if _, err := rtJobs.RunJob(ctx, Job{Kind: "testRun", Args: JobArgs{"feed_id": feed}}); err != nil {
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
		checkErr(t, rtJobs.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "test"} }))

		// Add jobs
		for _, feed := range feeds {
			if _, err := rtJobs.AddJob(ctx, Job{Kind: "test", Args: JobArgs{"feed_id": feed}}); err != nil {
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
		checkErr(t, rtJobs.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testAddJobs"} }))
		// Add jobs
		var jobs []Job
		for i := 0; i < 10; i++ {
			// 1 job: j=0
			jobs = append(jobs, Job{Kind: "testAddJobs", Args: JobArgs{"test": fmt.Sprintf("n:%d", i)}})
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
		checkErr(t, rtJobs.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testUnique"} }))
		checkErr(t, rtJobs.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testNotUnique"} }))

		// Add jobs
		for i := 0; i < 10; i++ {
			// 1 job: j=0
			for j := 0; j < 10; j++ {
				job := Job{Kind: "testUnique", Unique: true, Args: JobArgs{"test": fmt.Sprintf("n:%d", j/10)}}
				_, err := rtJobs.AddJob(ctx, job)
				checkErr(t, err)
			}
			// 3 jobs; j=3, j=6, j=9... j=0 is not unique
			for j := 0; j < 10; j++ {
				job := Job{Kind: "testUnique", Unique: true, Args: JobArgs{"test": fmt.Sprintf("n:%d", j/3)}}
				_, err := rtJobs.AddJob(ctx, job)
				checkErr(t, err)
			}
			// 10 jobs: j=0, j=0, j=2, j=2, j=4, j=4, j=6 j=6, j=8, j=8
			for j := 0; j < 10; j++ {
				job := Job{Kind: "testNotUnique", Args: JobArgs{"test": fmt.Sprintf("n:%d", j/2)}}
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
		checkErr(t, rtJobs.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testDeadline"} }))
		// Add jobs
		rtJobs.AddJob(ctx, Job{Kind: "testDeadline", Args: JobArgs{"test": "test"}, Deadline: 0})
		rtJobs.AddJob(ctx, Job{Kind: "testDeadline", Args: JobArgs{"test": "test"}, Deadline: time.Now().Add(1 * time.Hour).Unix()})
		rtJobs.AddJob(ctx, Job{Kind: "testDeadline", Args: JobArgs{"test": "test"}, Deadline: time.Now().Add(-1 * time.Hour).Unix()})
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
		checkErr(t, rtJobs.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testMiddleware"} }))
		rtJobs.AddJob(ctx, Job{Kind: "testMiddleware", Args: JobArgs{"mw": "ok1"}})
		rtJobs.AddJob(ctx, Job{Kind: "testMiddleware", Args: JobArgs{"mw": "ok2"}})
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
		checkErr(t, rtJobs.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testStatus"} }))
		st, err := rtJobs.AddJob(ctx, Job{Kind: "testStatus", UserID: "alice", Args: JobArgs{"x": "1"}})
		checkErr(t, err)
		assert.NotEmpty(t, st.Job.ID, "AddJob should assign a JobId")
		assert.Equal(t, "alice", st.Job.UserID)
		queuedSt, err := rtJobs.Status(ctx, st.Job.ID)
		checkErr(t, err)
		assert.Equal(t, st.Job.ID, queuedSt.Job.ID)
		go func() {
			time.Sleep(sleepyTime)
			rtJobs.Stop(ctx)
		}()
		if err := rtJobs.Run(ctx); err != nil {
			t.Fatal(err)
		}
		got, err := rtJobs.Status(ctx, st.Job.ID)
		checkErr(t, err)
		assert.True(t, got.State.Terminal(), "expected terminal state, got %s", got.State)
		assert.Equal(t, jobs.JobStateSucceeded, got.State)
		_, err = rtJobs.Status(ctx, "does-not-exist")
		assert.Error(t, err)
	})
	t.Run("watch", func(t *testing.T) {
		rtJobs := newQueue(queueName(t))
		count := int64(0)
		checkErr(t, rtJobs.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testWatch"} }))
		st, err := rtJobs.AddJob(ctx, Job{Kind: "testWatch", UserID: "alice", Args: JobArgs{"x": "live"}})
		checkErr(t, err)
		// Open the watch before the queue starts so we observe the full transition.
		ch, err := rtJobs.Watch(ctx, st.Job.ID)
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
		// Per-run-unique JobType so persistent backends don't see rows from prior runs.
		listType := fmt.Sprintf("testList-%d-%d", os.Getpid(), time.Now().UnixNano())
		count := int64(0)
		checkErr(t, rtJobs.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: listType} }))
		for i := 0; i < 5; i++ {
			_, err := rtJobs.AddJob(ctx, Job{Kind: listType, UserID: "alice", Args: JobArgs{"i": i}})
			checkErr(t, err)
			time.Sleep(1 * time.Millisecond)
		}
		for i := 0; i < 3; i++ {
			_, err := rtJobs.AddJob(ctx, Job{Kind: listType, UserID: "bob", Args: JobArgs{"i": i}})
			checkErr(t, err)
			time.Sleep(1 * time.Millisecond)
		}
		all, err := rtJobs.ListJobs(ctx, jobs.JobListOptions{Kind: listType})
		checkErr(t, err)
		assert.Equal(t, 8, len(all.Jobs))
		aliceCtx := userCtx("alice")
		mine, err := rtJobs.ListJobs(aliceCtx, jobs.JobListOptions{Kind: listType, UserID: "bob"})
		checkErr(t, err)
		assert.Equal(t, 5, len(mine.Jobs))
		for _, st := range mine.Jobs {
			assert.Equal(t, "alice", st.Job.UserID)
		}
		seen := map[string]bool{}
		var cursor string
		for pages := 0; pages < 10; pages++ {
			page, err := rtJobs.ListJobs(ctx, jobs.JobListOptions{Kind: listType, UserID: "alice", Limit: 2, After: cursor})
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
		rtJobs := newQueue(queueName(t))
		count := int64(0)
		checkErr(t, rtJobs.RegisterWorker(func() JobWorker { return &TestWorker{count: &count, kind: "testAuth"} }))
		st, err := rtJobs.AddJob(ctx, Job{Kind: "testAuth", UserID: "alice", Args: JobArgs{"x": "1"}})
		checkErr(t, err)
		if _, err := rtJobs.Status(userCtx("alice"), st.Job.ID); err != nil {
			t.Errorf("owner Status: %v", err)
		}
		if _, err := rtJobs.Status(ctx, st.Job.ID); err != nil {
			t.Errorf("admin Status: %v", err)
		}
		_, err = rtJobs.Status(userCtx("bob"), st.Job.ID)
		assert.ErrorIs(t, err, jobs.ErrJobAccessDenied)
		_, err = rtJobs.Watch(userCtx("bob"), st.Job.ID)
		assert.ErrorIs(t, err, jobs.ErrJobAccessDenied)
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
