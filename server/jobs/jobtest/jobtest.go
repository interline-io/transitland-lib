package jobtest

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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

func TestJobQueue(t *testing.T, newQueue func(string) JobQueue) {
	ctx := context.Background()
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
			if err := rtJobs.RunJob(ctx, Job{JobType: "testRun", JobArgs: JobArgs{"feed_id": feed}}); err != nil {
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
			if err := rtJobs.AddJob(ctx, Job{JobType: "test", JobArgs: JobArgs{"feed_id": feed}}); err != nil {
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
		checkErr(t, rtJobs.AddJobs(ctx, jobs))
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
				checkErr(t, rtJobs.AddJob(ctx, job))
			}
			// 3 jobs; j=3, j=6, j=9... j=0 is not unique
			for j := 0; j < 10; j++ {
				job := Job{JobType: "testUnique", Unique: true, JobArgs: JobArgs{"test": fmt.Sprintf("n:%d", j/3)}}
				checkErr(t, rtJobs.AddJob(ctx, job))
			}
			// 10 jobs: j=0, j=0, j=2, j=2, j=4, j=4, j=6 j=6, j=8, j=8
			for j := 0; j < 10; j++ {
				job := Job{JobType: "testNotUnique", JobArgs: JobArgs{"test": fmt.Sprintf("n:%d", j/2)}}
				checkErr(t, rtJobs.AddJob(ctx, job))
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
