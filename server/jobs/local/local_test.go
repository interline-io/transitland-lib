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
