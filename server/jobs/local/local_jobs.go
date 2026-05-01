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
)

func init() {
	var _ jobs.JobQueue = &LocalJobs{}
}

// watchBufferSize is overprovisioned so that the small number of events a
// single job emits (queued -> running -> terminal) never fill the channel and
// force a non-blocking drop. LocalJobs is a development adapter; corner cases
// where this still overflows are documented on Watch's contract — only Status
// is authoritative.
const watchBufferSize = 256
const defaultListLimit = 100
const defaultTerminalTTL = 1 * time.Hour
const sweepInterval = 1 * time.Minute

type jobEntry struct {
	mu       sync.Mutex
	status   jobs.JobStatus
	watchers []chan jobs.JobEvent
}

type LocalJobs struct {
	jobs           chan jobs.Job
	jobfuncs       []func(context.Context, jobs.Job) error
	running        bool
	middlewares    []jobs.JobMiddleware
	uniqueJobs     map[string]bool
	uniqueJobsLock sync.Mutex
	jobMapper      *jobs.JobMapper
	ctx            context.Context
	cancel         context.CancelFunc

	registryLock sync.Mutex
	registry     map[string]*jobEntry
	terminalTTL  time.Duration
}

func NewLocalJobs() *LocalJobs {
	f := &LocalJobs{
		jobs:        make(chan jobs.Job, 1000),
		uniqueJobs:  map[string]bool{},
		jobMapper:   jobs.NewJobMapper(),
		registry:    map[string]*jobEntry{},
		terminalTTL: defaultTerminalTTL,
	}
	return f
}

// SetTerminalTTL configures how long terminal job entries are retained before
// being evicted from the in-memory registry. Zero disables eviction. The
// eviction sweeper is started by Run; if Run is never called, no eviction
// happens regardless of TTL.
func (f *LocalJobs) SetTerminalTTL(d time.Duration) {
	f.terminalTTL = d
}

func (f *LocalJobs) Use(mwf jobs.JobMiddleware) {
	f.middlewares = append(f.middlewares, mwf)
}

func (f *LocalJobs) AddQueue(queue string, count int) error {
	for i := 0; i < count; i++ {
		f.jobfuncs = append(f.jobfuncs, f.runQueuedJob)
	}
	return nil
}

func (f *LocalJobs) AddJobType(jobFn jobs.JobFn) error {
	return f.jobMapper.AddJobType(jobFn)
}

func (f *LocalJobs) AddJobs(ctx context.Context, in []jobs.Job) ([]jobs.JobStatus, error) {
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

func (f *LocalJobs) AddJob(ctx context.Context, job jobs.Job) (jobs.JobStatus, error) {
	if f.jobs == nil {
		return jobs.JobStatus{}, errors.New("closed")
	}
	job.JobId = uuid.NewString()
	if job.Unique {
		key, err := job.HexKey()
		if err != nil {
			return jobs.JobStatus{}, err
		}
		f.uniqueJobsLock.Lock()
		if _, ok := f.uniqueJobs[key]; ok {
			f.uniqueJobsLock.Unlock()
			log.Trace().Interface("job", job).Msgf("already locked: %s", key)
			f.registerJob(job)
			f.updateStatus(job.JobId, jobs.JobStateCancelled, "duplicate")
			st, _ := f.statusUnchecked(job.JobId)
			return st, nil
		}
		f.uniqueJobs[key] = true
		f.uniqueJobsLock.Unlock()
		log.Trace().Interface("job", job).Msgf("locked: %s", key)
	}
	status := f.registerJob(job)
	f.jobs <- job
	return status, nil
}

func (f *LocalJobs) registerJob(job jobs.Job) jobs.JobStatus {
	now := time.Now().UTC()
	entry := &jobEntry{
		status: jobs.JobStatus{
			JobId:       job.JobId,
			UserId:      job.UserId,
			State:       jobs.JobStateQueued,
			Job:         job,
			SubmittedAt: now,
		},
	}
	f.registryLock.Lock()
	f.registry[job.JobId] = entry
	f.registryLock.Unlock()
	return entry.status
}

func (f *LocalJobs) getEntry(jobId string) *jobEntry {
	f.registryLock.Lock()
	defer f.registryLock.Unlock()
	return f.registry[jobId]
}

func (f *LocalJobs) statusUnchecked(jobId string) (jobs.JobStatus, error) {
	entry := f.getEntry(jobId)
	if entry == nil {
		return jobs.JobStatus{}, jobs.ErrJobNotFound
	}
	entry.mu.Lock()
	defer entry.mu.Unlock()
	return entry.status, nil
}

func (f *LocalJobs) Status(ctx context.Context, jobId string) (jobs.JobStatus, error) {
	st, err := f.statusUnchecked(jobId)
	if err != nil {
		return jobs.JobStatus{}, err
	}
	if err := jobs.CheckJobAccess(ctx, st); err != nil {
		return jobs.JobStatus{}, err
	}
	return st, nil
}

func (f *LocalJobs) Watch(ctx context.Context, jobId string) (<-chan jobs.JobEvent, error) {
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
			JobId:   jobId,
			State:   entry.status.State,
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

func (f *LocalJobs) cleanupWatcherOnCancel(ctx context.Context, entry *jobEntry, ch chan jobs.JobEvent) {
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

func (f *LocalJobs) ListJobs(ctx context.Context, opts jobs.JobListOptions) (jobs.JobListResult, error) {
	user := authn.ForContext(ctx)
	if user == nil {
		return jobs.JobListResult{}, jobs.ErrJobAccessDenied
	}
	isAdmin := user.HasRole("admin")
	filterUserId := opts.UserId
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
		if filterUserId != "" && st.UserId != filterUserId {
			continue
		}
		if opts.JobType != "" && st.Job.JobType != opts.JobType {
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
		return snapshot[i].JobId < snapshot[j].JobId
	})

	if opts.After != "" {
		cur, err := decodeCursor(opts.After)
		if err != nil {
			return jobs.JobListResult{}, err
		}
		// Keyset advance: find the first row strictly after the cursor under
		// (SubmittedAt desc, JobId asc). This is robust against eviction —
		// even if the cursor's row is gone, we still know where to resume.
		cutoff := -1
		for i, st := range snapshot {
			if st.SubmittedAt.Before(cur.SubmittedAt) ||
				(st.SubmittedAt.Equal(cur.SubmittedAt) && st.JobId > cur.JobId) {
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
// can resume even if the cursor's underlying row was evicted.
type listCursor struct {
	SubmittedAt time.Time `json:"t"`
	JobId       string    `json:"i"`
}

func encodeCursor(st jobs.JobStatus) string {
	b, _ := json.Marshal(listCursor{SubmittedAt: st.SubmittedAt, JobId: st.JobId})
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

func (f *LocalJobs) updateStatus(jobId string, state jobs.JobState, errMsg string) {
	entry := f.getEntry(jobId)
	if entry == nil {
		return
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
	evt := jobs.JobEvent{JobId: jobId, State: state, Message: errMsg, Time: now}
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
}

func (w *LocalJobs) AddPeriodicJob(ctx context.Context, jobFunc func() jobs.Job, period time.Duration, cronTab string) error {
	ticker := time.NewTicker(period)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				w.AddJob(ctx, jobFunc())
			}
		}
	}()
	return nil
}

// runQueuedJob is the worker-pool entry point. The job's JobId is already set
// and registered by AddJob, so we go straight to execution.
func (f *LocalJobs) runQueuedJob(ctx context.Context, job jobs.Job) error {
	_, err := f.execute(ctx, job)
	return err
}

// RunJob runs a job synchronously. JobId is always assigned by the adapter;
// any caller-set value is overwritten.
func (f *LocalJobs) RunJob(ctx context.Context, job jobs.Job) (jobs.JobStatus, error) {
	job.JobId = uuid.NewString()
	f.registerJob(job)
	return f.execute(ctx, job)
}

// execute runs the worker chain for an already-registered job, transitioning
// state through queued -> running -> succeeded/failed/cancelled.
func (f *LocalJobs) execute(ctx context.Context, job jobs.Job) (jobs.JobStatus, error) {
	now := time.Now().In(time.UTC).Unix()
	if job.JobDeadline > 0 && job.JobDeadline < now {
		log.Trace().Int64("job_deadline", job.JobDeadline).Int64("now", now).Msg("job skipped - deadline in past")
		f.updateStatus(job.JobId, jobs.JobStateCancelled, "deadline in past")
		st, _ := f.statusUnchecked(job.JobId)
		return st, nil
	}
	if job.Unique {
		key, err := job.HexKey()
		if err != nil {
			f.updateStatus(job.JobId, jobs.JobStateFailed, err.Error())
			st, _ := f.statusUnchecked(job.JobId)
			return st, err
		}
		f.uniqueJobsLock.Lock()
		delete(f.uniqueJobs, key)
		f.uniqueJobsLock.Unlock()
		log.Trace().Interface("job", job).Msgf("unlocked: %s", key)
	}
	w, err := f.jobMapper.GetRunner(job.JobType, job.JobArgs)
	if err != nil {
		f.updateStatus(job.JobId, jobs.JobStateFailed, err.Error())
		st, _ := f.statusUnchecked(job.JobId)
		return st, err
	}
	if w == nil {
		f.updateStatus(job.JobId, jobs.JobStateFailed, "no job")
		st, _ := f.statusUnchecked(job.JobId)
		return st, errors.New("no job")
	}
	for _, mwf := range f.middlewares {
		w = mwf(w, job)
		if w == nil {
			f.updateStatus(job.JobId, jobs.JobStateFailed, "no job")
			st, _ := f.statusUnchecked(job.JobId)
			return st, errors.New("no job")
		}
	}
	f.updateStatus(job.JobId, jobs.JobStateRunning, "")
	runErr := w.Run(ctx)
	if runErr != nil {
		f.updateStatus(job.JobId, jobs.JobStateFailed, runErr.Error())
	} else {
		f.updateStatus(job.JobId, jobs.JobStateSucceeded, "")
	}
	st, _ := f.statusUnchecked(job.JobId)
	return st, runErr
}

func (f *LocalJobs) Run(ctx context.Context) error {
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

func (f *LocalJobs) evictLoop(ctx context.Context) {
	ticker := time.NewTicker(sweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			f.evictTerminal(time.Now().UTC().Add(-f.terminalTTL))
		}
	}
}

// evictTerminal removes terminal-state entries whose FinishedAt is at or before cutoff.
func (f *LocalJobs) evictTerminal(cutoff time.Time) {
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

func (f *LocalJobs) Stop(ctx context.Context) error {
	if !f.running {
		return errors.New("not running")
	}
	close(f.jobs)
	f.cancel()
	f.running = false
	f.jobs = nil
	return nil
}
