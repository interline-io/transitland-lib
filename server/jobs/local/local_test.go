package jobs

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/jobs"
	"github.com/interline-io/transitland-lib/server/jobs/jobtest"
)

func TestLocalJobs(t *testing.T) {
	newQueue := func(queueName string) jobs.JobQueue {
		q := jobs.NewJobLogger(NewLocalJobs())
		q.AddQueue("default", 4)
		return q
	}
	jobtest.TestJobQueue(t, newQueue)
}
