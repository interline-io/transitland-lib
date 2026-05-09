package jobs

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/jobs"
	"github.com/interline-io/transitland-lib/server/jobs/jobtest"
)

func TestLocalBackend(t *testing.T) {
	newSetup := func(queueName string) jobtest.TestSetup {
		runner := jobs.NewRunner()
		backend := NewLocalBackend(runner, map[string]QueueOpts{
			queueName: {Workers: 4},
		})
		return jobtest.TestSetup{Runner: runner, Backend: backend, QueueName: queueName}
	}
	jobtest.TestBackend(t, newSetup)
}

// TestLocalBackendStress runs the heavy-load harness against LocalBackend.
// Skipped by default; set JOBSTRESS=1 to run.
func TestLocalBackendStress(t *testing.T) {
	newSetup := func(queueName string) jobtest.TestSetup {
		runner := jobs.NewRunner()
		// Disable terminal eviction so Status polling across the run still
		// finds the entries.
		backend := NewLocalBackend(runner, map[string]QueueOpts{
			queueName: {Workers: 8, TerminalTTL: -1},
		})
		return jobtest.TestSetup{Runner: runner, Backend: backend, QueueName: queueName}
	}
	jobtest.StressBackend(t, newSetup, jobtest.StressOpts{})
}
