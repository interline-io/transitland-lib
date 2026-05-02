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

func init() {
	var _ jobs.Backend = &LocalBackend{}
	var _ jobs.JobStatusReporter = &LocalBackend{}
	var _ jobs.PeriodicScheduler = &LocalBackend{}
}

// watchBufferSize is overprovisioned so that the small number of events a
// single job emits (queued -> running -> terminal) never fill the channel and
// force a non-blocking drop. LocalBackend is a development adapter; corner
// cases where this still overflows are documented on Watch's contract — only
// Status is authoritative.
const watchBufferSize = 256
const defaultListLimit = 100
const defaultTerminalTTL = 1 * time.Hour
const sweepInterval = 1 * time.Minute

type jobEntry struct {
	mu              sync.Mutex
	status          jobs.JobStatus
	watchers        []chan jobs.JobEvent
	cancelRun       context.CancelFunc // nil until execute starts; cancels the worker's ctx
	cancelRequested bool               // set by Cancel; queued jobs check this before running
}

type LocalBackend struct {
	runner         *jobs.Runner
	jobs           chan jobs.Job
	jobfuncs       []func(context.Context, jobs.Job) error
	running        bool
	uniqueJobs     map[string]time.Time // key -> expiresAt; zero means "until consumed by execute"
	uniqueJobsLock sync.Mutex
	ctx            context.Context
	cancel         context.CancelFunc

	registryLock sync.Mutex
	registry     map[string]*jobEntry
	terminalTTL  time.Duration

	periodicLock sync.Mutex
	periodics    map[string]context.CancelFunc
}

func NewLocalBackend(runner *jobs.Runner) *LocalBackend {
	return &LocalBackend{
		runner:      runner,
		jobs:        make(chan jobs.Job, 1000),
		uniqueJobs:  map[string]time.Time{},
		registry:    map[string]*jobEntry{},
		terminalTTL: defaultTerminalTTL,
		periodics:   map[string]context.CancelFunc{},
	}
}

// SetTerminalTTL configures how long terminal job entries are retained before
// being evicted from the in-memory registry. Zero disables eviction. The
// eviction sweeper is started by Run; if Run is never called, no eviction
// happens regardless of TTL.
func (f *LocalBackend) SetTerminalTTL(d time.Duration) {
	f.terminalTTL = d
}

func (f *LocalBackend) AddQueue(queue string, count int) error {
	for i := 0; i < count; i++ {
		f.jobfuncs = append(f.jobfuncs, f.runQueuedJob)
	}
	return nil
}

func (f *LocalBackend) AddJobs(ctx context.Context, in []jobs.Job) ([]jobs.JobStatus, error) {
	out := make([]jobs.JobStatus, 0, len(in))
	for _, job := range in {
		st, err := f.AddJob(ctx, job)
		if err != nil {
			return out, err
		}
		out = append(out, st)
	}
	return out, nil
}

func (f *LocalBackend) AddJob(ctx context.Context, job jobs.Job) (jobs.JobStatus, error) {
	if f.jobs == nil {
		return jobs.JobStatus{}, errors.New("closed")
	}
	job.ID = uuid.NewString()
	if job.Unique {
		key, err := job.HexKey()
		if err != nil {
			return jobs.JobStatus{}, err
		}
		f.uniqueJobsLock.Lock()
		if expiresAt, ok := f.uniqueJobs[key]; ok && (expiresAt.IsZero() || time.Now().Before(expiresAt)) {
			f.uniqueJobsLock.Unlock()
			log.Trace().Interface("job", job).Msgf("already locked: %s", key)
			f.registerJob(job)
			return f.updateStatus(job.ID, jobs.JobStateCancelled, "duplicate"), nil
		}
		// Either no entry, or the entry's window expired — claim the slot.
		var expiresAt time.Time
		if job.UniqueWindow > 0 {
			expiresAt = time.Now().Add(job.UniqueWindow)
		}
		f.uniqueJobs[key] = expiresAt
		f.uniqueJobsLock.Unlock()
		log.Trace().Interface("job", job).Msgf("locked: %s", key)
	}
	status := f.registerJob(job)
	f.jobs <- job
	return status, nil
}

func (f *LocalBackend) registerJob(job jobs.Job) jobs.JobStatus {
	now := time.Now().UTC()
	entry := &jobEntry{
		status: jobs.JobStatus{
			State:       jobs.JobStateQueued,
			Job:         job,
			SubmittedAt: now,
		},
	}
	f.registryLock.Lock()
	f.registry[job.ID] = entry
	f.registryLock.Unlock()
	return entry.status
}

func (f *LocalBackend) getEntry(jobId string) *jobEntry {
	f.registryLock.Lock()
	defer f.registryLock.Unlock()
	return f.registry[jobId]
}

func (f *LocalBackend) statusUnchecked(jobId string) (jobs.JobStatus, error) {
	entry := f.getEntry(jobId)
	if entry == nil {
		return jobs.JobStatus{}, jobs.ErrJobNotFound
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	return entry.status, nil
}

func (f *LocalBackend) Status(ctx context.Context, jobId string) (jobs.JobStatus, error) {
	st, err := f.statusUnchecked(jobId)
	if err != nil {
		return jobs.JobStatus{}, err
	}
	if err := jobs.CheckJobAccess(ctx, st); err != nil {
		return jobs.JobStatus{}, err
	}
	return st, nil
}

func (f *LocalBackend) Watch(ctx context.Context, jobId string) (<-chan jobs.JobEvent, error) {
	entry := f.getEntry(jobId)
	if entry == nil {
		return nil, jobs.ErrJobNotFound
	}
	entry.mu.Lock()
	if err := jobs.CheckJobAccess(ctx, entry.status); err != nil {
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
	go f.cleanupWatcherOnCancel(ctx, entry, ch)
	return ch, nil
}

func (f *LocalBackend) cleanupWatcherOnCancel(ctx context.Context, entry *jobEntry, ch chan jobs.JobEvent) {
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

func (f *LocalBackend) ListJobs(ctx context.Context, opts jobs.JobListOptions) (jobs.JobListResult, error) {
	user := authn.ForContext(ctx)
	if user == nil {
		return jobs.JobListResult{}, jobs.ErrJobAccessDenied
	}
	isAdmin := user.HasRole("admin")
	filterUserId := opts.UserID
	if !isAdmin {
		filterUserId = user.ID()
		if filterUserId == "" {
			return jobs.JobListResult{}, jobs.ErrJobAccessDenied
		}
	}
	stateSet := map[jobs.JobState]bool{}
	for _, s := range opts.States {
		stateSet[s] = true
	}

	f.registryLock.Lock()
	snapshot := make([]jobs.JobStatus, 0, len(f.registry))
	for _, entry := range f.registry {
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
	f.registryLock.Unlock()

	sort.Slice(snapshot, func(i, j int) bool {
		if !snapshot[i].SubmittedAt.Equal(snapshot[j].SubmittedAt) {
			return snapshot[i].SubmittedAt.After(snapshot[j].SubmittedAt)
		}
		return snapshot[i].Job.ID < snapshot[j].Job.ID
	})

	if opts.After != "" {
		cur, err := decodeCursor(opts.After)
		if err != nil {
			return jobs.JobListResult{}, err
		}
		// Keyset advance: find the first row strictly after the cursor under
		// (SubmittedAt desc, ID asc). This is robust against eviction —
		// even if the cursor's row is gone, we still know where to resume.
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
	return jobs.JobListResult{Jobs: snapshot, NextCursor: next}, nil
}

// listCursor is the keyset payload — both fields of the sort key, so paging
// can resume even if the cursor's underlying row was evicted. SubmittedAtNano
// is unix-nanos so the cursor doesn't depend on a particular time.Time JSON
// formatting and stays compact.
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

func (f *LocalBackend) updateStatus(jobId string, state jobs.JobState, errMsg string) jobs.JobStatus {
	entry := f.getEntry(jobId)
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

func (w *LocalBackend) AddPeriodicJob(ctx context.Context, jobFunc func() jobs.Job, period time.Duration, cronTab string) (string, error) {
	id := uuid.NewString()
	pctx, cancel := context.WithCancel(ctx)
	if cronTab != "" {
		schedule, err := cron.ParseStandard(cronTab)
		if err != nil {
			cancel()
			return "", err
		}
		w.registerPeriodic(id, cancel)
		go w.runCron(pctx, schedule, jobFunc)
		return id, nil
	}
	w.registerPeriodic(id, cancel)
	go w.runInterval(pctx, period, jobFunc)
	return id, nil
}

// RemovePeriodicJob stops a previously-added periodic schedule. Returns
// ErrJobNotFound if the id is unknown (or already removed).
func (w *LocalBackend) RemovePeriodicJob(ctx context.Context, id string) error {
	w.periodicLock.Lock()
	defer w.periodicLock.Unlock()
	cancel, ok := w.periodics[id]
	if !ok {
		return jobs.ErrJobNotFound
	}
	cancel()
	delete(w.periodics, id)
	return nil
}

func (w *LocalBackend) registerPeriodic(id string, cancel context.CancelFunc) {
	w.periodicLock.Lock()
	w.periodics[id] = cancel
	w.periodicLock.Unlock()
}

func (w *LocalBackend) runInterval(ctx context.Context, period time.Duration, jobFunc func() jobs.Job) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.AddJob(ctx, jobFunc())
		}
	}
}

func (w *LocalBackend) runCron(ctx context.Context, schedule cron.Schedule, jobFunc func() jobs.Job) {
	for {
		next := schedule.Next(time.Now())
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Until(next)):
			w.AddJob(ctx, jobFunc())
		}
	}
}

func (f *LocalBackend) runQueuedJob(ctx context.Context, job jobs.Job) error {
	_, err := f.execute(ctx, job)
	return err
}

func (f *LocalBackend) execute(ctx context.Context, job jobs.Job) (jobs.JobStatus, error) {
	// If Cancel was called while the job was queued, skip the run.
	entry := f.getEntry(job.ID)
	if entry != nil {
		entry.mu.Lock()
		cancelled := entry.cancelRequested
		entry.mu.Unlock()
		if cancelled {
			return f.updateStatus(job.ID, jobs.JobStateCancelled, "cancelled"), nil
		}
	}

	now := time.Now().In(time.UTC).Unix()
	if job.Deadline > 0 && job.Deadline < now {
		log.Trace().Int64("job_deadline", job.Deadline).Int64("now", now).Msg("job skipped - deadline in past")
		return f.updateStatus(job.ID, jobs.JobStateCancelled, "deadline in past"), nil
	}
	// Window=0 deduplicates only while a matching job is queued/running, so
	// release the lock when execution starts. Window>0 entries are time-based
	// and expire on their own.
	if job.Unique && job.UniqueWindow == 0 {
		key, err := job.HexKey()
		if err != nil {
			return f.updateStatus(job.ID, jobs.JobStateFailed, err.Error()), err
		}
		f.uniqueJobsLock.Lock()
		delete(f.uniqueJobs, key)
		f.uniqueJobsLock.Unlock()
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
	f.updateStatus(job.ID, jobs.JobStateRunning, "")
	runErr := f.runner.Run(runCtx, job)
	// If Cancel was called mid-run, treat as cancelled rather than failed.
	if entry != nil {
		entry.mu.Lock()
		cancelled := entry.cancelRequested
		entry.cancelRun = nil
		entry.mu.Unlock()
		if cancelled {
			return f.updateStatus(job.ID, jobs.JobStateCancelled, "cancelled"), nil
		}
	}
	if runErr != nil {
		return f.updateStatus(job.ID, jobs.JobStateFailed, runErr.Error()), runErr
	}
	return f.updateStatus(job.ID, jobs.JobStateSucceeded, ""), nil
}

// Cancel marks a queued or running job for cancellation. Idempotent on
// terminal jobs.
func (f *LocalBackend) Cancel(ctx context.Context, jobId string) error {
	entry := f.getEntry(jobId)
	if entry == nil {
		return jobs.ErrJobNotFound
	}
	entry.mu.Lock()
	if err := jobs.CheckJobAccess(ctx, entry.status); err != nil {
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
		cancelRun() // running job: ctx cancelled, execute will translate to JobStateCancelled
	}
	if wasQueued {
		// Mark immediately; the worker will see cancelRequested before starting.
		f.updateStatus(jobId, jobs.JobStateCancelled, "cancelled")
	}
	return nil
}

func (f *LocalBackend) Run(ctx context.Context) error {
	if f.running {
		return errors.New("already running")
	}
	f.ctx, f.cancel = context.WithCancel(ctx)
	f.running = true
	for _, jobfunc := range f.jobfuncs {
		go func(jf func(context.Context, jobs.Job) error) {
			for job := range f.jobs {
				jf(ctx, job)
			}
		}(jobfunc)
	}
	if f.terminalTTL > 0 {
		go f.evictLoop(f.ctx)
	}
	<-f.ctx.Done()
	return nil
}

func (f *LocalBackend) evictLoop(ctx context.Context) {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f.evictTerminal(time.Now().UTC().Add(-f.terminalTTL))
			f.sweepUniqueExpired(time.Now())
		}
	}
}

// sweepUniqueExpired drops UniqueWindow entries whose window has elapsed.
// Window-zero entries (cleared by execute) are left alone.
func (f *LocalBackend) sweepUniqueExpired(now time.Time) {
	f.uniqueJobsLock.Lock()
	defer f.uniqueJobsLock.Unlock()
	for key, expiresAt := range f.uniqueJobs {
		if !expiresAt.IsZero() && now.After(expiresAt) {
			delete(f.uniqueJobs, key)
		}
	}
}

// evictTerminal removes terminal-state entries whose FinishedAt is at or before cutoff.
func (f *LocalBackend) evictTerminal(cutoff time.Time) {
	f.registryLock.Lock()
	defer f.registryLock.Unlock()
	for id, entry := range f.registry {
		entry.mu.Lock()
		evict := entry.status.State.Terminal() &&
			entry.status.FinishedAt != nil &&
			!entry.status.FinishedAt.After(cutoff)
		entry.mu.Unlock()
		if evict {
			delete(f.registry, id)
		}
	}
}

func (f *LocalBackend) Stop(ctx context.Context) error {
	if !f.running {
		return errors.New("not running")
	}
	close(f.jobs)
	f.cancel()
	f.running = false
	f.jobs = nil
	return nil
}
