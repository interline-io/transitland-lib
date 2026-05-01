package jobs

import (
	"context"
	"encoding/base64"
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

const watchBufferSize = 16
const defaultListLimit = 100

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
}

func NewLocalJobs() *LocalJobs {
	f := &LocalJobs{
		jobs:       make(chan jobs.Job, 1000),
		uniqueJobs: map[string]bool{},
		jobMapper:  jobs.NewJobMapper(),
		registry:   map[string]*jobEntry{},
	}
	return f
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
	return ch, nil
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
		afterId, err := decodeCursor(opts.After)
		if err != nil {
			return jobs.JobListResult{}, err
		}
		idx := -1
		for i, st := range snapshot {
			if st.JobId == afterId {
				idx = i
				break
			}
		}
		if idx >= 0 {
			snapshot = snapshot[idx+1:]
		} else {
			snapshot = nil
		}
	}

	limit := opts.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	next := ""
	if len(snapshot) > limit {
		snapshot = snapshot[:limit]
		next = encodeCursor(snapshot[len(snapshot)-1].JobId)
	}
	return jobs.JobListResult{Jobs: snapshot, NextCursor: next}, nil
}

func encodeCursor(jobId string) string {
	return base64.URLEncoding.EncodeToString([]byte(jobId))
}

func decodeCursor(cursor string) (string, error) {
	b, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return "", errors.New("invalid cursor")
	}
	return string(b), nil
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

// runQueuedJob is the worker-pool adapter for the channel's func(ctx, job) error signature.
func (f *LocalJobs) runQueuedJob(ctx context.Context, job jobs.Job) error {
	_, err := f.RunJob(ctx, job)
	return err
}

func (f *LocalJobs) RunJob(ctx context.Context, job jobs.Job) (jobs.JobStatus, error) {
	if job.JobId == "" {
		job.JobId = uuid.NewString()
	}
	if f.getEntry(job.JobId) == nil {
		f.registerJob(job)
	}

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
	<-f.ctx.Done()
	return nil
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
