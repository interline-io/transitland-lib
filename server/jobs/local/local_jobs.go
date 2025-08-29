package jobs

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/jobs"
)

func init() {
	var _ jobs.JobQueue = &LocalJobs{}
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
}

func NewLocalJobs() *LocalJobs {
	f := &LocalJobs{
		jobs:       make(chan jobs.Job, 1000),
		uniqueJobs: map[string]bool{},
		jobMapper:  jobs.NewJobMapper(),
	}
	return f
}

func (f *LocalJobs) Use(mwf jobs.JobMiddleware) {
	f.middlewares = append(f.middlewares, mwf)
}

func (f *LocalJobs) AddQueue(queue string, count int) error {
	for i := 0; i < count; i++ {
		f.jobfuncs = append(f.jobfuncs, f.RunJob)
	}
	return nil
}

func (f *LocalJobs) AddJobType(jobFn jobs.JobFn) error {
	return f.jobMapper.AddJobType(jobFn)
}

func (f *LocalJobs) AddJobs(ctx context.Context, jobs []jobs.Job) error {
	for _, job := range jobs {
		err := f.AddJob(ctx, job)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *LocalJobs) AddJob(ctx context.Context, job jobs.Job) error {
	if f.jobs == nil {
		return errors.New("closed")
	}
	if job.Unique {
		f.uniqueJobsLock.Lock()
		defer f.uniqueJobsLock.Unlock()
		key, err := job.HexKey()
		if err != nil {
			return err
		}
		if _, ok := f.uniqueJobs[key]; ok {
			log.Trace().Interface("job", job).Msgf("already locked: %s", key)
			return nil
		} else {
			f.uniqueJobs[key] = true
			log.Trace().Interface("job", job).Msgf("locked: %s", key)
		}
	}
	f.jobs <- job
	return nil
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

func (f *LocalJobs) RunJob(ctx context.Context, job jobs.Job) error {
	now := time.Now().In(time.UTC).Unix()
	if job.JobDeadline > 0 && job.JobDeadline < now {
		log.Trace().Int64("job_deadline", job.JobDeadline).Int64("now", now).Msg("job skipped - deadline in past")
		return nil
	}
	if job.Unique {
		f.uniqueJobsLock.Lock()
		defer f.uniqueJobsLock.Unlock()
		key, err := job.HexKey()
		if err != nil {
			return err
		}
		delete(f.uniqueJobs, key)
		log.Trace().Interface("job", job).Msgf("unlocked: %s", key)
	}
	w, err := f.jobMapper.GetRunner(job.JobType, job.JobArgs)
	if err != nil {
		return err
	}
	if w == nil {
		return errors.New("no job")
	}
	for _, mwf := range f.middlewares {
		w = mwf(w, job)
		if w == nil {
			return errors.New("no job")
		}
	}
	return w.Run(ctx)
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
