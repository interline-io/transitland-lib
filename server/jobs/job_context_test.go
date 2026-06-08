package jobs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type captureWorker struct{ got *string }

func (c *captureWorker) Kind() string { return "capture" }
func (c *captureWorker) Run(ctx context.Context) error {
	*c.got = JobIDFromContext(ctx)
	return nil
}

// TestRunnerInjectsJobID confirms Runner.Run makes the job ID available to the
// worker context (the propagation that artifact creation relies on).
func TestRunnerInjectsJobID(t *testing.T) {
	var got string
	r := NewRunner()
	if err := r.Register(func() Worker { return &captureWorker{got: &got} }); err != nil {
		t.Fatal(err)
	}
	err := r.Run(context.Background(), Job{ID: "job-xyz", Kind: "capture"})
	assert.NoError(t, err)
	assert.Equal(t, "job-xyz", got)
}

func TestJobIDFromContextEmpty(t *testing.T) {
	assert.Equal(t, "", JobIDFromContext(context.Background()))
}
