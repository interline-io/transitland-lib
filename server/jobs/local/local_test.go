package jobs

import (
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/server/jobs"
	"github.com/interline-io/transitland-lib/server/jobs/jobtest"
)

func TestLocalBackend(t *testing.T) {
	newSetup := func(queueName string) jobtest.TestSetup {
		runner := jobs.NewRunner()
		backend := NewLocalBackend(runner, map[string]QueueOpts{
			queueName: {Workers: 4},
		}, nil)
		return jobtest.TestSetup{Runner: runner, Backend: backend, QueueName: queueName}
	}
	jobtest.TestBackend(t, newSetup)
}

// TestLocalBackendStress runs the heavy-load harness against LocalBackend.
// In-memory queue can take much heavier load than the harness defaults; turn
// the dials up here. Skipped by default; set JOBSTRESS=1 to run.
func TestLocalBackendStress(t *testing.T) {
	newSetup := func(queueName string) jobtest.TestSetup {
		runner := jobs.NewRunner()
		backend := NewLocalBackend(runner, map[string]QueueOpts{
			queueName: {Workers: 16},
		}, nil)
		return jobtest.TestSetup{Runner: runner, Backend: backend, QueueName: queueName}
	}
	jobtest.StressBackend(t, newSetup, jobtest.StressOpts{
		SubmitN:        100_000,
		SubmitWorkers:  50,
		FanoutSeeds:    8,
		FanoutChildren: 4,
		FanoutDepth:    5, // 1365 nodes × 8 seeds = 10,920 jobs
		CancelN:        1000,
		CancelSleep:    100 * time.Millisecond,
		WatchersPerJob: 10,
		WatchJobs:      100, // 1000 watcher channels
		UniqueAttempts: 1000,
		Timeout:        180 * time.Second,
	})
}
