package jobs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/jobs"
	"github.com/robfig/cron/v3"
)

// QueueOpts is per-queue configuration handed to NewLocalBackend.
type QueueOpts struct {
	// Workers is the number of goroutines draining the queue. Defaults to 1
	// if zero or negative.
	Workers int
	// TerminalTTL controls how long terminal-state job entries are retained
	// before being evicted from the in-memory registry. Zero = default (1h).
	// Negative = eviction disabled (useful for tests).
	TerminalTTL time.Duration
}

const (
	defaultListLimit   = 100
	defaultTerminalTTL = 1 * time.Hour
	sweepInterval      = 1 * time.Minute
	// watchBufferSize is overprovisioned so the small number of events a
	// single job emits (queued -> running -> terminal) never fills the
	// channel and forces a non-blocking drop. LocalBackend is a development
	// adapter; corner cases where this overflows are documented on Watch's
	// contract — only Status is authoritative.
	watchBufferSize = 256
)

// LocalBackend is a development backend that runs jobs in-process via
// goroutine pools. Each declared queue gets its own pool, registry, and
// dedup map. All queues share the supplied Runner.
type LocalBackend struct {
	runner  *jobs.Runner
	queues  map[string]*localQueue
	started bool
	cancel  context.CancelFunc
}

func init() {
	var _ jobs.Backend = &LocalBackend{}
	var _ jobs.Queue = &localQueue{}
	var _ jobs.StatusQueue = &localQueue{}
	var _ jobs.PeriodicQueue = &localQueue{}
}

// NewLocalBackend constructs a LocalBackend hosting the named queues. Each
// queue's worker pool is sized by QueueOpts.Workers.
func NewLocalBackend(runner *jobs.Runner, queues map[string]QueueOpts) *LocalBackend {
	b := &LocalBackend{
		runner: runner,
		queues: map[string]*localQueue{},
	}
	for name, opts := range queues {
		b.queues[name] = newLocalQueue(name, opts, runner)
	}
	return b
}

// Queue returns the per-queue handle for name, or nil if not declared.
func (b *LocalBackend) Queue(name string) jobs.Queue {
	q, ok := b.queues[name]
	if !ok {
		return nil
	}
	return q
}

// Run starts every queue's worker pool and eviction loop, then blocks until
// Stop is called (or the parent ctx is cancelled).
func (b *LocalBackend) Run(ctx context.Context) error {
	if b.started {
		return errors.New("already running")
	}
	b.started = true
	ctx, b.cancel = context.WithCancel(ctx)
	var wg sync.WaitGroup
	for _, q := range b.queues {
		q := q
		wg.Add(1)
		go func() {
			defer wg.Done()
			q.run(ctx)
		}()
	}
	<-ctx.Done()
	for _, q := range b.queues {
		q.shutdown()
	}
	wg.Wait()
	return nil
}

func (b *LocalBackend) Stop(ctx context.Context) error {
	if !b.started {
		return errors.New("not running")
	}
	b.cancel()
	b.started = false
	return nil
}

// localQueue is the per-queue handle returned by LocalBackend.Queue.
// Implements jobs.Queue, jobs.StatusQueue, jobs.PeriodicQueue.
type localQueue struct {
	name        string
	workers     int
	terminalTTL time.Duration
	runner      *jobs.Runner

	jobs    chan jobs.Job
	closeMu sync.Mutex
	closed  bool

	uniqueMu   sync.Mutex
	uniqueJobs map[string]time.Time

	registryMu sync.Mutex
	registry   map[string]*jobEntry

	periodicMu sync.Mutex
	periodics  map[string]context.CancelFunc
}

type jobEntry struct {
	mu              sync.Mutex
	status          jobs.JobStatus
	watchers        []chan jobs.JobEvent
	cancelRun       context.CancelFunc // nil until execute starts; cancels the worker's ctx
	cancelRequested bool               // set by Cancel; queued jobs check this before running
}

func newLocalQueue(name string, opts QueueOpts, runner *jobs.Runner) *localQueue {
	workers := opts.Workers
	if workers <= 0 {
		workers = 1
	}
	ttl := opts.TerminalTTL
	switch {
	case ttl < 0:
		ttl = 0 // disabled
	case ttl == 0:
		ttl = defaultTerminalTTL
	}
	return &localQueue{
		name:        name,
		workers:     workers,
		terminalTTL: ttl,
		runner:      runner,
		jobs:        make(chan jobs.Job, 1000),
		uniqueJobs:  map[string]time.Time{},
		registry:    map[string]*jobEntry{},
		periodics:   map[string]context.CancelFunc{},
	}
}

// run drains the queue with `workers` goroutines and sweeps terminal entries
// + expired unique locks until ctx is cancelled.
func (q *localQueue) run(ctx context.Context) {
	var wg sync.WaitGroup
	for i := 0; i < q.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range q.jobs {
				_, _ = q.execute(ctx, job)
			}
		}()
	}
	if q.terminalTTL > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			q.sweepLoop(ctx)
		}()
	}
	wg.Wait()
}

func (q *localQueue) shutdown() {
	q.closeMu.Lock()
	defer q.closeMu.Unlock()
	if q.closed {
		return
	}
	q.closed = true
	close(q.jobs)
}

func (q *localQueue) Submit(ctx context.Context, job jobs.Job) (jobs.JobStatus, error) {
	q.closeMu.Lock()
	if q.closed {
		q.closeMu.Unlock()
		return jobs.JobStatus{}, errors.New("closed")
	}
	q.closeMu.Unlock()
	job.ID = uuid.NewString()
	if job.Unique {
		key, err := job.HexKey()
		if err != nil {
			return jobs.JobStatus{}, err
		}
		q.uniqueMu.Lock()
		if expiresAt, ok := q.uniqueJobs[key]; ok && (expiresAt.IsZero() || time.Now().Before(expiresAt)) {
			q.uniqueMu.Unlock()
			log.Trace().Interface("job", job).Msgf("already locked: %s", key)
			q.registerJob(job)
			return q.updateStatus(job.ID, jobs.JobStateCancelled, "duplicate"), nil
		}
		// Either no entry, or the entry's window expired — claim the slot.
		var expiresAt time.Time
		if job.UniqueWindow > 0 {
			expiresAt = time.Now().Add(job.UniqueWindow)
		}
		q.uniqueJobs[key] = expiresAt
		q.uniqueMu.Unlock()
		log.Trace().Interface("job", job).Msgf("locked: %s", key)
	}
	status := q.registerJob(job)
	q.jobs <- job
	return status, nil
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
	if err := jobs.CheckAccess(ctx, st); err != nil {
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
	if err := jobs.CheckAccess(ctx, entry.status); err != nil {
		entry.mu.Unlock()
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
		entry.mu.Unlock()
		return ch, nil
	}
	entry.watchers = append(entry.watchers, ch)
	entry.mu.Unlock()
	// Drop the registration if the caller's context cancels before terminal.
	// Both this goroutine and updateStatus take entry.mu, so we won't double-close.
	go q.cleanupWatcherOnCancel(ctx, entry, ch)
	return ch, nil
}

func (q *localQueue) cleanupWatcherOnCancel(ctx context.Context, entry *jobEntry, ch chan jobs.JobEvent) {
	<-ctx.Done()
	entry.mu.Lock()
	defer entry.mu.Unlock()
	for i, c := range entry.watchers {
		if c == ch {
			entry.watchers = append(entry.watchers[:i], entry.watchers[i+1:]...)
			close(c)
			return
		}
	}
	// Not found: updateStatus already broadcast and closed it on terminal.
}

func (q *localQueue) List(ctx context.Context, opts jobs.ListOptions) (jobs.ListResult, error) {
	user := authn.ForContext(ctx)
	if user == nil {
		return jobs.ListResult{}, jobs.ErrJobAccessDenied
	}
	isAdmin := user.HasRole("admin")
	filterUserId := opts.UserID
	if !isAdmin {
		filterUserId = user.ID()
		if filterUserId == "" {
			return jobs.ListResult{}, jobs.ErrJobAccessDenied
		}
	}
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
		if filterUserId != "" && st.Job.UserID != filterUserId {
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

	if opts.After != "" {
		cur, err := decodeCursor(opts.After)
		if err != nil {
			return jobs.ListResult{}, err
		}
		// Keyset advance: find the first row strictly after the cursor under
		// (SubmittedAt desc, ID asc). Robust against eviction — even if the
		// cursor's row is gone, we still know where to resume.
		cutoff := -1
		for i, st := range snapshot {
			ts := st.SubmittedAt.UnixNano()
			if ts < cur.SubmittedAtNano ||
				(ts == cur.SubmittedAtNano && st.Job.ID > cur.ID) {
				cutoff = i
				break
			}
		}
		if cutoff < 0 {
			snapshot = nil
		} else {
			snapshot = snapshot[cutoff:]
		}
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	next := ""
	if len(snapshot) > limit {
		snapshot = snapshot[:limit]
		next = encodeCursor(snapshot[len(snapshot)-1])
	}
	return jobs.ListResult{Jobs: snapshot, NextCursor: next}, nil
}

// listCursor is the keyset payload — both fields of the sort key, so paging
// can resume even if the cursor's underlying row was evicted. SubmittedAtNano
// stays compact and time.Time-format-independent.
type listCursor struct {
	SubmittedAtNano int64  `json:"t"`
	ID              string `json:"i"`
}

func encodeCursor(st jobs.JobStatus) string {
	b, _ := json.Marshal(listCursor{SubmittedAtNano: st.SubmittedAt.UnixNano(), ID: st.Job.ID})
	return base64.URLEncoding.EncodeToString(b)
}

func decodeCursor(cursor string) (listCursor, error) {
	raw, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return listCursor{}, errors.New("invalid cursor")
	}
	var c listCursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return listCursor{}, errors.New("invalid cursor")
	}
	return c, nil
}

func (q *localQueue) Cancel(ctx context.Context, jobId string) error {
	entry := q.getEntry(jobId)
	if entry == nil {
		return jobs.ErrJobNotFound
	}
	entry.mu.Lock()
	if err := jobs.CheckAccess(ctx, entry.status); err != nil {
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
	for _, ch := range entry.watchers {
		select {
		case ch <- evt:
		default:
		}
	}
	if state.Terminal() {
		for _, ch := range entry.watchers {
			close(ch)
		}
		entry.watchers = nil
	}
	return entry.status
}

func (q *localQueue) execute(ctx context.Context, job jobs.Job) (jobs.JobStatus, error) {
	// If Cancel was called while queued, skip the run.
	entry := q.getEntry(job.ID)
	if entry != nil {
		entry.mu.Lock()
		cancelled := entry.cancelRequested
		entry.mu.Unlock()
		if cancelled {
			return q.updateStatus(job.ID, jobs.JobStateCancelled, "cancelled"), nil
		}
	}

	now := time.Now().In(time.UTC).Unix()
	if job.Deadline > 0 && job.Deadline < now {
		log.Trace().Int64("job_deadline", job.Deadline).Int64("now", now).Msg("job skipped - deadline in past")
		return q.updateStatus(job.ID, jobs.JobStateCancelled, "deadline in past"), nil
	}
	// Window=0 deduplicates only while a matching job is queued/running, so
	// release the lock when execution starts. Window>0 entries are time-based
	// and expire on their own.
	if job.Unique && job.UniqueWindow == 0 {
		key, err := job.HexKey()
		if err != nil {
			return q.updateStatus(job.ID, jobs.JobStateFailed, err.Error()), err
		}
		q.uniqueMu.Lock()
		delete(q.uniqueJobs, key)
		q.uniqueMu.Unlock()
		log.Trace().Interface("job", job).Msgf("unlocked: %s", key)
	}
	// Derive a per-run ctx that Cancel can interrupt independently of Run/Stop.
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
	id := uuid.NewString()
	pctx, cancel := context.WithCancel(ctx)
	if cronTab != "" {
		schedule, err := cron.ParseStandard(cronTab)
		if err != nil {
			cancel()
			return "", err
		}
		q.registerPeriodic(id, cancel)
		go q.runCron(pctx, schedule, jobFunc)
		return id, nil
	}
	q.registerPeriodic(id, cancel)
	go q.runInterval(pctx, period, jobFunc)
	return id, nil
}

// RemovePeriodic stops a previously-added schedule. Returns ErrJobNotFound
// if the id is unknown (or already removed).
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

func (q *localQueue) sweepLoop(ctx context.Context) {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			q.evictTerminal(time.Now().UTC().Add(-q.terminalTTL))
			q.sweepUniqueExpired(time.Now())
		}
	}
}

// sweepUniqueExpired drops UniqueWindow entries whose window has elapsed.
// Window-zero entries (cleared by execute) are left alone.
func (q *localQueue) sweepUniqueExpired(now time.Time) {
	q.uniqueMu.Lock()
	defer q.uniqueMu.Unlock()
	for key, expiresAt := range q.uniqueJobs {
		if !expiresAt.IsZero() && now.After(expiresAt) {
			delete(q.uniqueJobs, key)
		}
	}
}

// evictTerminal removes terminal-state entries whose FinishedAt is at or before cutoff.
func (q *localQueue) evictTerminal(cutoff time.Time) {
	q.registryMu.Lock()
	defer q.registryMu.Unlock()
	for id, entry := range q.registry {
		entry.mu.Lock()
		evict := entry.status.State.Terminal() &&
			entry.status.FinishedAt != nil &&
			!entry.status.FinishedAt.After(cutoff)
		entry.mu.Unlock()
		if evict {
			delete(q.registry, id)
		}
	}
}
