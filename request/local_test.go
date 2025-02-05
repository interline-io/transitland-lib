package request

import (
	"context"
	"path/filepath"
	"testing"
)

func TestLocal(t *testing.T) {
	ctx := context.TODO()
	b := &Local{filepath.Join(t.TempDir(), "local")}
	testBucket(t, ctx, b)
}
