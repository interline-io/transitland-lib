package jobs

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/jobs"
	"github.com/robfig/cron/v3"
)

type QueueOpts struct {
	// Workers is the number of goroutines draining the queue. Defaults to 1.
	Workers int
}

const (
	defaultListLimit = 100
	// Overprovisioned so a job's few events never overflow; StatusQueue.Watch
	// documents the non-blocking-drop contract.
	watchBufferSize = 256
)

var (
	_ jobs.Backend       = (*LocalBackend)(nil)
	_ jobs.Queue         = (*localQueue)(nil)
	_ jobs.StatusQueue   = (*localQueue)(nil)
	_ jobs.PeriodicQueue = (*localQueue)(nil)
)

// LocalBackend is a development backend that runs jobs in-process via
// per-queue goroutine pools sharing one Runner.
type LocalBackend struct {
	runner  *jobs.Runner
	queues  map[string]*localQueue
	policy  jobs.AccessPolicy
	startMu sync.Mutex
	started bool
	cancel  context.CancelFunc
	runDone chan struct{}
	// drain + drained: Shutdown closes drain; drained gates the close under
	// startMu so Shutdown is idempotent. Run selects on drain or ctx.Done.
	drain   chan struct{}
	drained bool
}

// NewLocalBackend constructs an in-process backend. nil policy → CreatorOrAdmin.
func NewLocalBackend(runner *jobs.Runner, queues map[string]QueueOpts, policy jobs.AccessPolicy) *LocalBackend {
	if policy == nil {
		policy = jobs.CreatorOrAdmin{}
	}
	b := &LocalBackend{
		runner: runner,
		queues: map[string]*localQueue{},
		policy: policy,
	}
	for name, opts := range queues {
		b.queues[name] = newLocalQueue(name, opts, runner, policy)
	}
	return b
}

func (b *LocalBackend) Queue(name string) (jobs.Queue, error) {
	q, ok := b.queues[name]
	if !ok {
		return nil, jobs.ErrUnknownQueue
	}
	return q, nil
}

func (b *LocalBackend) Run(ctx context.Context) error {
	b.startMu.Lock()
	if b.started {
		b.startMu.Unlock()
		return errors.New("already running")
	}
	b.started = true
	ctx, b.cancel = context.WithCancel(ctx)
	b.runDone = make(chan struct{})
	b.drain = make(chan struct{})
	b.drained = false
	drain := b.drain
	b.startMu.Unlock()

	defer close(b.runDone)
	var wg sync.WaitGroup
	for _, q := range b.queues {
		q := q
		wg.Add(1)
		go func() {
			defer wg.Done()
			q.run(ctx)
		}()
	}
	// Either branch closes q.done below. For graceful drain, ctx stays alive
	// so in-flight worker.Run finishes naturally; hard stop cancels everything.
	select {
	case <-ctx.Done():
	case <-drain:
	}
	for _, q := range b.queues {
		q.shutdown()
	}
	wg.Wait()
	return nil
}

// Shutdown triggers graceful drain (queues stop accepting; in-flight workers
// run to completion with ctx alive) and blocks until done or ctx fires.
// Idempotent. Returns nil if Run was never called.
func (b *LocalBackend) Shutdown(ctx context.Context) error {
	b.startMu.Lock()
	done := b.runDone
	if done == nil {
		b.startMu.Unlock()
		return nil
	}
	if !b.drained && b.drain != nil {
		close(b.drain)
		b.drained = true
	}
	b.startMu.Unlock()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (b *LocalBackend) Stop(ctx context.Context) error {
	b.startMu.Lock()
	if !b.started {
		b.startMu.Unlock()
		return errors.New("not running")
	}
	cancel := b.cancel
	b.started = false
	b.startMu.Unlock()
	cancel()
	return nil
}

// Wait blocks until Run returns or ctx fires. Does not trigger shutdown.
// Returns nil if Run was never called.
func (b *LocalBackend) Wait(ctx context.Context) error {
	b.startMu.Lock()
	done := b.runDone
	b.startMu.Unlock()
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type localQueue struct {
	name    string
	workers int
	runner  *jobs.Runner
	policy  jobs.AccessPolicy

	jobs chan jobs.Job
	// done is closed on shutdown. q.jobs is intentionally not closed —
	// closing it would race with in-flight Submit sends.
	done     chan struct{}
	doneOnce sync.Once

	uniqueMu   sync.Mutex
	uniqueJobs map[string]struct{}

	registryMu sync.Mutex
	registry   map[string]*jobEntry

	periodicMu sync.Mutex
	periodics  map[string]context.CancelFunc
}

type jobEntry struct {
	mu              sync.Mutex
	status          jobs.JobStatus
	watchers        []chan jobs.JobEvent
	cancelRun       context.CancelFunc // set by execute; cancels just this worker's ctx
	cancelRequested bool               // queued jobs check this before running
}

func newLocalQueue(name string, opts QueueOpts, runner *jobs.Runner, policy jobs.AccessPolicy) *localQueue {
	workers := opts.Workers
	if workers <= 0 {
		workers = 1
	}
	return &localQueue{
		name:    name,
		workers: workers,
		runner:  runner,
		policy:  policy,
		// Large buffer so worker-spawns-children doesn't deadlock on a full
		// channel. Proper fix is an unbounded queue.
		jobs:       make(chan jobs.Job, 100_000),
		done:       make(chan struct{}),
		uniqueJobs: map[string]struct{}{},
		registry:   map[string]*jobEntry{},
		periodics:  map[string]context.CancelFunc{},
	}
}

func (q *localQueue) run(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < q.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-q.done:
					return
				case <-ctx.Done():
					return
				case job := <-q.jobs:
					_, _ = q.execute(ctx, job)
				}
			}
		}()
	}
	wg.Wait()
}

func (q *localQueue) shutdown() {
	q.doneOnce.Do(func() { close(q.done) })
}

func (q *localQueue) Submit(ctx context.Context, job jobs.Job) (jobs.JobStatus, error) {
	if err := q.policy.CanSubmit(ctx, job); err != nil {
		return jobs.JobStatus{}, err
	}
	// Best-effort fast path; the authoritative shutdown guard is the select below.
	select {
	case <-q.done:
		return jobs.JobStatus{}, errors.New("closed")
	default:
	}
	job.ID = uuid.NewString()
	if job.Opts.Unique {
		key, err := dedupKey(job.Kind, job.Args)
		if err != nil {
			return jobs.JobStatus{}, err
		}
		q.uniqueMu.Lock()
		if _, ok := q.uniqueJobs[key]; ok {
			q.uniqueMu.Unlock()
			q.registerJob(job)
			return q.updateStatus(job.ID, jobs.JobStateCancelled, "duplicate"), nil
		}
		q.uniqueJobs[key] = struct{}{}
		q.uniqueMu.Unlock()
	}
	status := q.registerJob(job)
	select {
	case q.jobs <- job:
		return status, nil
	case <-q.done:
		q.cleanupFailedSubmit(job)
		return jobs.JobStatus{}, errors.New("closed")
	case <-ctx.Done():
		q.cleanupFailedSubmit(job)
		return jobs.JobStatus{}, ctx.Err()
	}
}

// cleanupFailedSubmit reverses registry + dedup-key state when enqueue fails.
func (q *localQueue) cleanupFailedSubmit(job jobs.Job) {
	q.registryMu.Lock()
	delete(q.registry, job.ID)
	q.registryMu.Unlock()
	if job.Opts.Unique {
		key, err := dedupKey(job.Kind, job.Args)
		if err == nil {
			q.uniqueMu.Lock()
			delete(q.uniqueJobs, key)
			q.uniqueMu.Unlock()
		}
	}
}

func dedupKey(kind string, args jobs.Args) (string, error) {
	bytes, err := json.Marshal(args)
	if err != nil {
		return "", err
	}
	sum := sha1.Sum(bytes)
	return kind + ":" + hex.EncodeToString(sum[:]), nil
}

func (q *localQueue) SubmitMany(ctx context.Context, in []jobs.Job) ([]jobs.JobStatus, error) {
	out := make([]jobs.JobStatus, 0, len(in))
	for _, job := range in {
		st, err := q.Submit(ctx, job)
		if err != nil {
			return out, err
		}
		out = append(out, st)
	}
	return out, nil
}

func (q *localQueue) registerJob(job jobs.Job) jobs.JobStatus {
	now := time.Now().UTC()
	entry := &jobEntry{
		status: jobs.JobStatus{
			State:       jobs.JobStateQueued,
			Job:         job,
			SubmittedAt: now,
		},
	}
	q.registryMu.Lock()
	q.registry[job.ID] = entry
	q.registryMu.Unlock()
	return entry.status
}

func (q *localQueue) getEntry(jobId string) *jobEntry {
	q.registryMu.Lock()
	defer q.registryMu.Unlock()
	return q.registry[jobId]
}

func (q *localQueue) Status(ctx context.Context, jobId string) (jobs.JobStatus, error) {
	entry := q.getEntry(jobId)
	if entry == nil {
		return jobs.JobStatus{}, jobs.ErrJobNotFound
	}
	entry.mu.Lock()
	st := entry.status
	entry.mu.Unlock()
	if err := q.policy.CanRead(ctx, st); err != nil {
		return jobs.JobStatus{}, err
	}
	return st, nil
}

func (q *localQueue) Watch(ctx context.Context, jobId string) (<-chan jobs.JobEvent, error) {
	entry := q.getEntry(jobId)
	if entry == nil {
		return nil, jobs.ErrJobNotFound
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	if err := q.policy.CanRead(ctx, entry.status); err != nil {
		return nil, err
	}
	ch := make(chan jobs.JobEvent, watchBufferSize)
	if entry.status.State.Terminal() {
		ch <- jobs.JobEvent{
			JobID:   jobId,
			State:   entry.status.State,
			Attempt: entry.status.Attempt,
			Message: entry.status.Error,
			Time:    time.Now().UTC(),
		}
		close(ch)
		return ch, nil
	}
	entry.watchers = append(entry.watchers, ch)
	return ch, nil
}

func (q *localQueue) List(ctx context.Context, opts jobs.ListOptions) (jobs.ListResult, error) {
	scoped, err := q.policy.ScopeList(ctx, opts)
	if err != nil {
		return jobs.ListResult{}, err
	}
	opts = scoped
	filterUserId := opts.UserID
	stateSet := map[jobs.JobState]bool{}
	for _, s := range opts.States {
		stateSet[s] = true
	}

	q.registryMu.Lock()
	snapshot := make([]jobs.JobStatus, 0, len(q.registry))
	for _, entry := range q.registry {
		entry.mu.Lock()
		st := entry.status
		entry.mu.Unlock()
		if filterUserId != "" && st.Job.Opts.UserID != filterUserId {
			continue
		}
		if opts.Kind != "" && st.Job.Kind != opts.Kind {
			continue
		}
		if len(stateSet) > 0 && !stateSet[st.State] {
			continue
		}
		snapshot = append(snapshot, st)
	}
	q.registryMu.Unlock()

	sort.Slice(snapshot, func(i, j int) bool {
		if !snapshot[i].SubmittedAt.Equal(snapshot[j].SubmittedAt) {
			return snapshot[i].SubmittedAt.After(snapshot[j].SubmittedAt)
		}
		return snapshot[i].Job.ID < snapshot[j].Job.ID
	})

	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}
	if offset >= len(snapshot) {
		return jobs.ListResult{}, nil
	}
	snapshot = snapshot[offset:]

	limit := opts.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if len(snapshot) > limit {
		snapshot = snapshot[:limit]
	}
	return jobs.ListResult{Jobs: snapshot}, nil
}

func (q *localQueue) Cancel(ctx context.Context, jobId string) error {
	entry := q.getEntry(jobId)
	if entry == nil {
		return jobs.ErrJobNotFound
	}
	entry.mu.Lock()
	if err := q.policy.CanRead(ctx, entry.status); err != nil {
		entry.mu.Unlock()
		return err
	}
	if entry.status.State.Terminal() {
		entry.mu.Unlock()
		return nil
	}
	entry.cancelRequested = true
	cancelRun := entry.cancelRun
	wasQueued := entry.status.State == jobs.JobStateQueued
	entry.mu.Unlock()

	if cancelRun != nil {
		cancelRun()
	}
	if wasQueued {
		q.updateStatus(jobId, jobs.JobStateCancelled, "cancelled")
	}
	return nil
}

func (q *localQueue) updateStatus(jobId string, state jobs.JobState, errMsg string) jobs.JobStatus {
	entry := q.getEntry(jobId)
	if entry == nil {
		return jobs.JobStatus{}
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	now := time.Now().UTC()
	entry.status.State = state
	if state == jobs.JobStateRunning && entry.status.StartedAt == nil {
		t := now
		entry.status.StartedAt = &t
	}
	if state.Terminal() {
		t := now
		entry.status.FinishedAt = &t
	}
	if errMsg != "" {
		entry.status.Error = errMsg
	}
	evt := jobs.JobEvent{JobID: jobId, State: state, Attempt: entry.status.Attempt, Message: errMsg, Time: now}
	terminal := state.Terminal()
	for _, ch := range entry.watchers {
		select {
		case ch <- evt:
		default:
		}
		if terminal {
			close(ch)
		}
	}
	if terminal {
		entry.watchers = nil
	}
	return entry.status
}

func (q *localQueue) execute(ctx context.Context, job jobs.Job) (jobs.JobStatus, error) {
	// Defer registered before any early-return so cancelled/deadline-past
	// paths still release the key that Submit acquired.
	if job.Opts.Unique {
		key, err := dedupKey(job.Kind, job.Args)
		if err != nil {
			// Same Args succeeded in Submit; defensive only.
			return q.updateStatus(job.ID, jobs.JobStateFailed, err.Error()), err
		}
		defer func() {
			q.uniqueMu.Lock()
			delete(q.uniqueJobs, key)
			q.uniqueMu.Unlock()
		}()
	}

	entry := q.getEntry(job.ID)
	if entry != nil {
		entry.mu.Lock()
		cancelled := entry.cancelRequested
		entry.mu.Unlock()
		if cancelled {
			return q.updateStatus(job.ID, jobs.JobStateCancelled, "cancelled"), nil
		}
	}

	if !job.Opts.Deadline.IsZero() && time.Now().UTC().After(job.Opts.Deadline) {
		log.Trace().Time("job_deadline", job.Opts.Deadline).Msg("job skipped - deadline in past")
		return q.updateStatus(job.ID, jobs.JobStateCancelled, "deadline in past"), nil
	}
	// Per-run ctx so Cancel can interrupt this worker without Stopping the backend.
	runCtx, runCancel := context.WithCancel(ctx)
	defer runCancel()
	if entry != nil {
		entry.mu.Lock()
		entry.cancelRun = runCancel
		entry.mu.Unlock()
	}
	q.updateStatus(job.ID, jobs.JobStateRunning, "")
	runErr := q.runner.Run(runCtx, job)
	if entry != nil {
		entry.mu.Lock()
		cancelled := entry.cancelRequested
		entry.cancelRun = nil
		entry.mu.Unlock()
		if cancelled {
			return q.updateStatus(job.ID, jobs.JobStateCancelled, "cancelled"), nil
		}
	}
	if runErr != nil {
		return q.updateStatus(job.ID, jobs.JobStateFailed, runErr.Error()), runErr
	}
	return q.updateStatus(job.ID, jobs.JobStateSucceeded, ""), nil
}

func (q *localQueue) AddPeriodic(ctx context.Context, jobFunc func() jobs.Job, period time.Duration, cronTab string) (string, error) {
	if cronTab == "" && period <= 0 {
		return "", errors.New("AddPeriodic: either cronTab or positive period required")
	}
	var schedule cron.Schedule
	if cronTab != "" {
		s, err := cron.ParseStandard(cronTab)
		if err != nil {
			return "", err
		}
		schedule = s
	}
	id := uuid.NewString()
	pctx, cancel := context.WithCancel(ctx)
	// Cancel on backend shutdown OR caller ctx; drop the map entry on exit
	// so caller-cancelled periodics don't leak.
	go func() {
		select {
		case <-q.done:
			cancel()
		case <-pctx.Done():
		}
		q.periodicMu.Lock()
		delete(q.periodics, id)
		q.periodicMu.Unlock()
	}()
	q.registerPeriodic(id, cancel)
	if schedule != nil {
		go q.runCron(pctx, schedule, jobFunc)
	} else {
		go q.runInterval(pctx, period, jobFunc)
	}
	return id, nil
}

func (q *localQueue) RemovePeriodic(ctx context.Context, id string) error {
	q.periodicMu.Lock()
	defer q.periodicMu.Unlock()
	cancel, ok := q.periodics[id]
	if !ok {
		return jobs.ErrJobNotFound
	}
	cancel()
	delete(q.periodics, id)
	return nil
}

func (q *localQueue) registerPeriodic(id string, cancel context.CancelFunc) {
	q.periodicMu.Lock()
	q.periodics[id] = cancel
	q.periodicMu.Unlock()
}

func (q *localQueue) runInterval(ctx context.Context, period time.Duration, jobFunc func() jobs.Job) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			q.Submit(ctx, jobFunc())
		}
	}
}

func (q *localQueue) runCron(ctx context.Context, schedule cron.Schedule, jobFunc func() jobs.Job) {
	for {
		next := schedule.Next(time.Now())
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(next)):
			q.Submit(ctx, jobFunc())
		}
	}
}
