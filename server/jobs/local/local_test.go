package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/jobs"
	"github.com/interline-io/transitland-lib/server/jobs/jobtest"
	"github.com/stretchr/testify/assert"
)

func TestLocalJobs(t *testing.T) {
	newQueue := func(queueName string) jobs.JobQueue {
		q := jobs.NewJobLogger(NewLocalJobs())
		q.AddQueue("default", 4)
		return q
	}
	jobtest.TestJobQueue(t, newQueue)
}

type noopWorker struct{}

func (n *noopWorker) Kind() string                  { return "noop" }
func (n *noopWorker) Run(ctx context.Context) error { return nil }

func TestLocalJobsEvictTerminal(t *testing.T) {
	q := NewLocalJobs()
	q.SetTerminalTTL(time.Hour)
	if err := q.AddJobType(func() jobs.JobWorker { return &noopWorker{} }); err != nil {
		t.Fatal(err)
	}
	admin := authn.NewCtxUser("admin", "", "").WithRoles("admin")
	ctx := authn.WithUser(context.Background(), admin)

	st, err := q.RunJob(ctx, jobs.Job{JobType: "noop", UserId: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	// Pre-cutoff: still present.
	q.evictTerminal(time.Now().UTC().Add(-2 * time.Hour))
	if _, err := q.Status(ctx, st.JobId); err != nil {
		t.Fatalf("status before eviction: %v", err)
	}
	// Cutoff in the future: terminal entry is past it; gets evicted.
	q.evictTerminal(time.Now().UTC().Add(time.Hour))
	_, err = q.Status(ctx, st.JobId)
	assert.ErrorIs(t, err, jobs.ErrJobNotFound)
}

