package jobtest

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/server/jobs"
)

// StressOpts tunes the stress scenarios. Zero values fall back to defaults.
type StressOpts struct {
	// Throughput
	SubmitN       int // total jobs to submit in the throughput scenario
	SubmitWorkers int // goroutines submitting concurrently

	// Fanout: spawner -> children, recursively to depth.
	// Total jobs = sum_{i=0..Depth} Children^i
	FanoutSeeds    int // number of root spawner jobs
	FanoutChildren int // children per spawner
	FanoutDepth    int // recursion depth (0 = leaf only)

	// Cancellation under load
	CancelN     int           // jobs submitted; half are cancelled
	CancelSleep time.Duration // per-job sleep to keep them in flight

	// Concurrent watchers
	WatchersPerJob int // watchers attached per job in the watch scenario
	WatchJobs      int // number of jobs in the watch scenario

	// Unique-dedup contention
	UniqueAttempts int // goroutines racing to submit the same Unique job

	// Timeout caps each scenario.
	Timeout time.Duration
}

func (o *StressOpts) defaults() {
	if o.SubmitN == 0 {
		o.SubmitN = 1000
	}
	if o.SubmitWorkers == 0 {
		o.SubmitWorkers = 10
	}
	if o.FanoutSeeds == 0 {
		o.FanoutSeeds = 4
	}
	if o.FanoutChildren == 0 {
		o.FanoutChildren = 4
	}
	if o.FanoutDepth == 0 {
		o.FanoutDepth = 3
	}
	if o.CancelN == 0 {
		o.CancelN = 100
	}
	if o.CancelSleep == 0 {
		o.CancelSleep = 200 * time.Millisecond
	}
	if o.WatchersPerJob == 0 {
		o.WatchersPerJob = 5
	}
	if o.WatchJobs == 0 {
		o.WatchJobs = 20
	}
	if o.UniqueAttempts == 0 {
		o.UniqueAttempts = 50
	}
	if o.Timeout == 0 {
		o.Timeout = 60 * time.Second
	}
}

// StressBackend is an opt-in heavy-load conformance suite. Set JOBSTRESS=1 to
// run; otherwise skips. Tunable via StressOpts (zero = defaults). Skips
// scenarios that depend on capabilities the queue doesn't implement
// (StatusQueue for cancel/watch, etc.).
//
// Backends that aren't durable (LocalBackend) sweep terminal entries on a
// background timer; pass TerminalTTL: -1 in queue opts so the harness can
// poll Status across a long run without seeing entries evicted.
func StressBackend(t *testing.T, newSetup func(string) TestSetup, opts StressOpts) {
	if os.Getenv("JOBSTRESS") == "" {
		t.Skip("set JOBSTRESS=1 to run")
	}
	opts.defaults()

	t.Run("throughput", func(t *testing.T) { stressThroughput(t, newSetup, opts) })
	t.Run("fanout", func(t *testing.T) { stressFanout(t, newSetup, opts) })
	t.Run("cancel-under-load", func(t *testing.T) { stressCancel(t, newSetup, opts) })
	t.Run("concurrent-watchers", func(t *testing.T) { stressWatchers(t, newSetup, opts) })
	t.Run("unique-contention", func(t *testing.T) { stressUnique(t, newSetup, opts) })
}

// counterWorker increments a shared atomic when it runs. Used by throughput
// and unique-contention scenarios.
type counterWorker struct {
	N int64 `json:"n"`

	kind   string
	count  *int64
	durMin time.Duration
	durMax time.Duration
}

func (w *counterWorker) Kind() string { return w.kind }
func (w *counterWorker) Run(ctx context.Context) error {
	if w.durMax > 0 {
		d := w.durMin
		if w.durMax > w.durMin {
			d += time.Duration(int64(w.N) % int64(w.durMax-w.durMin))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(d):
		}
	}
	atomic.AddInt64(w.count, 1)
	return nil
}

// spawnerWorker enqueues Children copies of itself with Depth-1, until
// Depth reaches 0. Used to exercise jobs-spawning-jobs.
type spawnerWorker struct {
	Depth    int    `json:"depth"`
	Children int    `json:"children"`
	Mark     string `json:"mark"`

	kind   string
	queue  jobs.Queue
	ran    *int64
	failed *int64
}

func (w *spawnerWorker) Kind() string { return w.kind }
func (w *spawnerWorker) Run(ctx context.Context) error {
	atomic.AddInt64(w.ran, 1)
	if w.Depth <= 0 {
		return nil
	}
	for i := 0; i < w.Children; i++ {
		child := jobs.Job{
			Kind: w.kind,
			Args: jobs.Args{
				"depth":    w.Depth - 1,
				"children": w.Children,
				"mark":     w.Mark,
			},
		}
		if _, err := w.queue.Submit(ctx, child); err != nil {
			atomic.AddInt64(w.failed, 1)
			return err
		}
	}
	return nil
}

func stressThroughput(t *testing.T, newSetup func(string) TestSetup, opts StressOpts) {
	setup := newSetup(uniqueQueueName(t))
	var count int64
	const kind = "stress-throughput"
	checkErr(t, setup.Runner.Register(func() jobs.Worker {
		return &counterWorker{kind: kind, count: &count}
	}))

	q := setup.Queue()
	ctx := adminCtx()

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = setup.Backend.Run(runCtx) }()

	start := time.Now()
	var wg sync.WaitGroup
	per := opts.SubmitN / opts.SubmitWorkers
	for w := 0; w < opts.SubmitWorkers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for i := 0; i < per; i++ {
				_, err := q.Submit(ctx, jobs.Job{Kind: kind, Args: jobs.Args{"i": w*per + i}})
				if err != nil {
					t.Errorf("submit: %v", err)
					return
				}
			}
		}(w)
	}
	wg.Wait()
	submitDur := time.Since(start)
	expected := int64(per * opts.SubmitWorkers)

	if !waitFor(opts.Timeout, func() bool { return atomic.LoadInt64(&count) >= expected }) {
		t.Fatalf("throughput: only %d/%d ran after %s", atomic.LoadInt64(&count), expected, opts.Timeout)
	}
	totalDur := time.Since(start)
	_ = setup.Backend.Stop(runCtx)

	t.Logf("throughput: %d submits in %s (%.0f/s); %d completions in %s (%.0f/s)",
		expected, submitDur, float64(expected)/submitDur.Seconds(),
		atomic.LoadInt64(&count), totalDur, float64(atomic.LoadInt64(&count))/totalDur.Seconds())
}

func stressFanout(t *testing.T, newSetup func(string) TestSetup, opts StressOpts) {
	setup := newSetup(uniqueQueueName(t))
	q := setup.Queue()
	ctx := adminCtx()

	var ran, failed int64
	const kind = "stress-spawner"
	checkErr(t, setup.Runner.Register(func() jobs.Worker {
		return &spawnerWorker{kind: kind, queue: q, ran: &ran, failed: &failed}
	}))

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = setup.Backend.Run(runCtx) }()

	start := time.Now()
	for i := 0; i < opts.FanoutSeeds; i++ {
		_, err := q.Submit(ctx, jobs.Job{
			Kind: kind,
			Args: jobs.Args{
				"depth":    opts.FanoutDepth,
				"children": opts.FanoutChildren,
				"mark":     fmt.Sprintf("seed-%d", i),
			},
		})
		checkErr(t, err)
	}

	// Tree size: 1 + c + c^2 + ... + c^d nodes per seed
	expectedPerSeed := geomSum(opts.FanoutChildren, opts.FanoutDepth)
	expected := int64(opts.FanoutSeeds * expectedPerSeed)

	if !waitFor(opts.Timeout, func() bool { return atomic.LoadInt64(&ran) >= expected }) {
		t.Fatalf("fanout: only %d/%d ran after %s (failed submits: %d)",
			atomic.LoadInt64(&ran), expected, opts.Timeout, atomic.LoadInt64(&failed))
	}
	dur := time.Since(start)
	_ = setup.Backend.Stop(runCtx)

	t.Logf("fanout: %d seeds × tree(c=%d,d=%d)=%d → %d ran in %s (%.0f/s); %d submit-failures",
		opts.FanoutSeeds, opts.FanoutChildren, opts.FanoutDepth, expectedPerSeed,
		atomic.LoadInt64(&ran), dur, float64(atomic.LoadInt64(&ran))/dur.Seconds(),
		atomic.LoadInt64(&failed))
}

func stressCancel(t *testing.T, newSetup func(string) TestSetup, opts StressOpts) {
	setup := newSetup(uniqueQueueName(t))
	q := setup.Queue()
	sq, ok := q.(jobs.StatusQueue)
	if !ok {
		t.Skipf("queue %T does not implement StatusQueue", q)
	}
	ctx := adminCtx()

	var ran int64
	const kind = "stress-cancel"
	checkErr(t, setup.Runner.Register(func() jobs.Worker {
		return &counterWorker{kind: kind, count: &ran, durMin: opts.CancelSleep, durMax: opts.CancelSleep}
	}))

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = setup.Backend.Run(runCtx) }()

	ids := make([]string, 0, opts.CancelN)
	for i := 0; i < opts.CancelN; i++ {
		st, err := q.Submit(ctx, jobs.Job{Kind: kind, Args: jobs.Args{"i": i}})
		checkErr(t, err)
		ids = append(ids, st.Job.ID)
	}

	// Cancel half, in parallel.
	var wg sync.WaitGroup
	cancelTarget := opts.CancelN / 2
	for i := 0; i < cancelTarget; i++ {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			_ = sq.Cancel(ctx, id)
		}(ids[i])
	}
	wg.Wait()

	// Wait for everything to reach terminal.
	if !waitFor(opts.Timeout, func() bool {
		terminal := 0
		for _, id := range ids {
			st, err := sq.Status(ctx, id)
			if err == nil && st.State.Terminal() {
				terminal++
			}
		}
		return terminal == len(ids)
	}) {
		t.Fatalf("cancel: not all jobs terminal after %s", opts.Timeout)
	}

	cancelled, succeeded, failed, other := 0, 0, 0, 0
	for _, id := range ids {
		st, err := sq.Status(ctx, id)
		if err != nil {
			other++
			continue
		}
		switch st.State {
		case jobs.JobStateCancelled:
			cancelled++
		case jobs.JobStateSucceeded:
			succeeded++
		case jobs.JobStateFailed:
			failed++
		default:
			other++
		}
	}
	_ = setup.Backend.Stop(runCtx)

	t.Logf("cancel: %d submitted, %d cancel calls; cancelled=%d succeeded=%d failed=%d other=%d ran-counter=%d",
		opts.CancelN, cancelTarget, cancelled, succeeded, failed, other, atomic.LoadInt64(&ran))
	if cancelled == 0 {
		t.Errorf("cancel: expected at least some jobs cancelled, got 0")
	}
}

func stressWatchers(t *testing.T, newSetup func(string) TestSetup, opts StressOpts) {
	setup := newSetup(uniqueQueueName(t))
	q := setup.Queue()
	sq, ok := q.(jobs.StatusQueue)
	if !ok {
		t.Skipf("queue %T does not implement StatusQueue", q)
	}
	ctx := adminCtx()

	var ran int64
	const kind = "stress-watch"
	checkErr(t, setup.Runner.Register(func() jobs.Worker {
		return &counterWorker{kind: kind, count: &ran, durMin: 50 * time.Millisecond, durMax: 50 * time.Millisecond}
	}))

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = setup.Backend.Run(runCtx) }()

	ids := make([]string, 0, opts.WatchJobs)
	for i := 0; i < opts.WatchJobs; i++ {
		st, err := q.Submit(ctx, jobs.Job{Kind: kind, Args: jobs.Args{"i": i}})
		checkErr(t, err)
		ids = append(ids, st.Job.ID)
	}

	var wg sync.WaitGroup
	closed := int64(0)
	for _, id := range ids {
		for w := 0; w < opts.WatchersPerJob; w++ {
			wg.Add(1)
			go func(id string) {
				defer wg.Done()
				ch, err := sq.Watch(ctx, id)
				if err != nil {
					t.Errorf("watch %s: %v", id, err)
					return
				}
				for range ch { // drain until close
				}
				atomic.AddInt64(&closed, 1)
			}(id)
		}
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(opts.Timeout):
		t.Fatalf("watch: only %d/%d watchers closed after %s",
			atomic.LoadInt64(&closed), opts.WatchJobs*opts.WatchersPerJob, opts.Timeout)
	}
	_ = setup.Backend.Stop(runCtx)

	t.Logf("watch: %d jobs × %d watchers = %d closed cleanly",
		opts.WatchJobs, opts.WatchersPerJob, atomic.LoadInt64(&closed))
}

func stressUnique(t *testing.T, newSetup func(string) TestSetup, opts StressOpts) {
	setup := newSetup(uniqueQueueName(t))
	q := setup.Queue()
	ctx := adminCtx()

	var ran int64
	const kind = "stress-unique"
	checkErr(t, setup.Runner.Register(func() jobs.Worker {
		return &counterWorker{kind: kind, count: &ran, durMin: 100 * time.Millisecond, durMax: 100 * time.Millisecond}
	}))

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = setup.Backend.Run(runCtx) }()

	// Race many goroutines submitting the same Unique job.
	mark := strconv.FormatInt(time.Now().UnixNano(), 36)
	var wg sync.WaitGroup
	submitted := int64(0)
	for i := 0; i < opts.UniqueAttempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := q.Submit(ctx, jobs.Job{
				Kind:   kind,
				Unique: true,
				Args:   jobs.Args{"mark": mark},
			})
			if err == nil {
				atomic.AddInt64(&submitted, 1)
			}
		}()
	}
	wg.Wait()

	// Wait for the one accepted job to run.
	if !waitFor(opts.Timeout, func() bool { return atomic.LoadInt64(&ran) >= 1 }) {
		t.Fatalf("unique: job never ran (submitted=%d)", atomic.LoadInt64(&submitted))
	}
	// Give a small grace window for any duplicate to slip through.
	time.Sleep(200 * time.Millisecond)
	_ = setup.Backend.Stop(runCtx)

	got := atomic.LoadInt64(&ran)
	t.Logf("unique: %d submit attempts (%d returned ok), %d actually ran",
		opts.UniqueAttempts, atomic.LoadInt64(&submitted), got)
	if got != 1 {
		t.Errorf("unique: expected exactly 1 run, got %d", got)
	}
}

// waitFor polls cond until true or timeout. Returns whether cond ever became true.
func waitFor(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return cond()
}

// geomSum returns 1 + c + c^2 + ... + c^d.
func geomSum(c, d int) int {
	sum, term := 0, 1
	for i := 0; i <= d; i++ {
		sum += term
		term *= c
	}
	return sum
}
