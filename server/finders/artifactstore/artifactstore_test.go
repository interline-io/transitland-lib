package artifactstore

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeFilename(t *testing.T) {
	cases := []struct{ in, want string }{
		{"export.zip", "export.zip"},
		{"my report (1).csv", "my_report__1_.csv"},
		{"../../etc/passwd", "passwd"},
		{"a/b/c.txt", "c.txt"},
		{`a\b\c.txt`, "c.txt"},
		{"..", ""},
		{".", ""},
		{".hidden", "hidden"},
		{"  spaced.txt  ", "spaced.txt"},
		{"", ""},
		{"weird\n\tname.json", "weird__name.json"},
	}
	for _, tc := range cases {
		assert.Equalf(t, tc.want, sanitizeFilename(tc.in), "sanitizeFilename(%q)", tc.in)
	}
}

// TestStoreRoundTrip exercises the full create -> storage + row -> read path.
// Requires a Postgres test DB (with the tl_job_artifacts migration applied);
// skips otherwise.
func TestStoreRoundTrip(t *testing.T) {
	if msg, ok := testutil.CheckTestDB(); !ok {
		t.Skip(msg)
	}
	db := testutil.MustOpenTestDB(t)
	ctx := context.Background()
	dir := t.TempDir()
	store := NewStore(db, dir)

	const jobID = "artifactstore-test-job"
	// Make counts deterministic across reruns.
	_, _ = db.ExecContext(ctx, "DELETE FROM tl_job_artifacts WHERE job_id = $1", jobID)
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM tl_job_artifacts WHERE job_id = $1", jobID)
	})

	content := "hello, artifacts"
	sum := sha1.Sum([]byte(content))
	wantSHA := hex.EncodeToString(sum[:])

	sc := store.For(jobID, "alice", "test")
	art, err := sc.CreateReader(ctx, model.ArtifactOpts{Filename: "out.txt", ContentType: "text/plain"}, strings.NewReader(content))
	require.NoError(t, err)
	assert.Greater(t, art.ID, 0)
	assert.Equal(t, jobID, art.JobID)
	assert.Equal(t, "alice", art.UserID)
	assert.Equal(t, "test", art.JobKind)
	assert.Equal(t, "out.txt", art.Filename)
	assert.Equal(t, "text/plain", art.ContentType)
	assert.Equal(t, int64(len(content)), art.SizeBytes)
	assert.Equal(t, wantSHA, art.SHA1)
	assert.True(t, strings.HasPrefix(art.StorageKey, "job-artifacts/"+jobID+"/"), "key=%s", art.StorageKey)

	// Blob is on disk with the expected content.
	blob, err := os.ReadFile(filepath.Join(dir, art.StorageKey))
	require.NoError(t, err)
	assert.Equal(t, content, string(blob))

	// ListByJob + GetByID.
	list, err := store.ListByJob(ctx, jobID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, art.ID, list[0].ID)

	got, err := store.GetByID(ctx, art.ID)
	require.NoError(t, err)
	assert.Equal(t, art.StorageKey, got.StorageKey)

	_, err = store.GetByID(ctx, 0x7fffffff)
	assert.ErrorIs(t, err, model.ErrArtifactNotFound)

	// Empty job id (e.g. fire-and-forget backend) is rejected.
	_, err = store.For("", "", "").CreateReader(ctx, model.ArtifactOpts{Filename: "x"}, strings.NewReader("x"))
	assert.Error(t, err)
}
