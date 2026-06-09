package artifactstore

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
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
		// Header-injection chars (the reason this function exists): CR/LF, ';',
		// '=' must never survive into a Content-Disposition value.
		{"file\r\nSet-Cookie: x.zip", "file__Set-Cookie__x.zip"},
		{"file;name.csv", "file_name.csv"},
		{"file=value.txt", "file_value.txt"},
	}
	for _, tc := range cases {
		assert.Equalf(t, tc.want, sanitizeFilename(tc.in), "sanitizeFilename(%q)", tc.in)
	}
}

// TestCreateWithoutStorageErrors confirms a write fails when no artifact
// storage is configured — there is no silent fallback to other storage. Needs
// no DB: the guard returns before any database use.
func TestCreateWithoutStorageErrors(t *testing.T) {
	sc := NewStore(nil, "").For("job-1", "alice", "test")
	_, err := sc.CreateReader(context.Background(), model.ArtifactOpts{Filename: "out.txt"}, strings.NewReader("x"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no artifact storage configured")
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

	// The three persisted artifacts the subtests assert against. Created up front
	// (not inside subtests) so a failure here stops the run before the dependent
	// subtests, and so ListByJob below sees the full set.
	art, err := sc.CreateReader(ctx, model.ArtifactOpts{Filename: "out.txt", ContentType: "text/plain"}, strings.NewReader(content))
	require.NoError(t, err)
	srcPath := filepath.Join(t.TempDir(), "src.txt")
	require.NoError(t, os.WriteFile(srcPath, []byte(content), 0o600))
	fileArt, err := sc.CreateFile(ctx, model.ArtifactOpts{Filename: "file.txt", ContentType: "text/plain"}, srcPath)
	require.NoError(t, err)
	defArt, err := sc.CreateReader(ctx, model.ArtifactOpts{Filename: "default.bin", ContentType: "  "}, strings.NewReader("z"))
	require.NoError(t, err)

	t.Run("CreateReader populates the row", func(t *testing.T) {
		assert.Greater(t, art.ID, 0)
		assert.Equal(t, jobID, art.JobID)
		assert.Equal(t, "alice", art.UserID)
		assert.Equal(t, "test", art.JobKind)
		assert.Equal(t, "out.txt", art.Filename)
		assert.Equal(t, "text/plain", art.ContentType)
		assert.Equal(t, int64(len(content)), art.SizeBytes)
		assert.Equal(t, wantSHA, art.SHA1)
	})

	t.Run("storage key is job-artifacts/<jobID>/<uuid>/<filename>", func(t *testing.T) {
		// Parse the uuid segment rather than prefix-check: it's what makes keys
		// collision-free and unguessable, and a prefix check would miss its loss.
		keyParts := strings.Split(art.StorageKey, "/")
		require.Len(t, keyParts, 4, "key=%s", art.StorageKey)
		assert.Equal(t, "job-artifacts", keyParts[0])
		assert.Equal(t, jobID, keyParts[1])
		_, err := uuid.Parse(keyParts[2])
		assert.NoError(t, err, "middle segment should be a uuid: %q", keyParts[2])
		assert.Equal(t, "out.txt", keyParts[3])
	})

	t.Run("blob written to disk", func(t *testing.T) {
		blob, err := os.ReadFile(filepath.Join(dir, art.StorageKey))
		require.NoError(t, err)
		assert.Equal(t, content, string(blob))
	})

	t.Run("GetByID", func(t *testing.T) {
		got, err := store.GetByID(ctx, art.ID)
		require.NoError(t, err)
		assert.Equal(t, art.StorageKey, got.StorageKey)
		_, err = store.GetByID(ctx, 0x7fffffff)
		assert.ErrorIs(t, err, model.ErrArtifactNotFound)
	})

	t.Run("CreateFile streams from a file on disk", func(t *testing.T) {
		assert.Equal(t, int64(len(content)), fileArt.SizeBytes)
		assert.Equal(t, wantSHA, fileArt.SHA1)
		blob, err := os.ReadFile(filepath.Join(dir, fileArt.StorageKey))
		require.NoError(t, err)
		assert.Equal(t, content, string(blob))
	})

	t.Run("CreateFile with a missing source errors before any row", func(t *testing.T) {
		_, err := sc.CreateFile(ctx, model.ArtifactOpts{Filename: "missing.txt"}, filepath.Join(dir, "does-not-exist"))
		assert.Error(t, err)
	})

	t.Run("blank ContentType defaults to octet-stream", func(t *testing.T) {
		assert.Equal(t, "application/octet-stream", defArt.ContentType)
	})

	t.Run("ListByJob returns newest first", func(t *testing.T) {
		list, err := store.ListByJob(ctx, jobID)
		require.NoError(t, err)
		require.Len(t, list, 3)
		assert.Equal(t, defArt.ID, list[0].ID)
		assert.Equal(t, fileArt.ID, list[1].ID)
		assert.Equal(t, art.ID, list[2].ID)
	})

	t.Run("empty job id is rejected", func(t *testing.T) {
		_, err := store.For("", "", "").CreateReader(ctx, model.ArtifactOpts{Filename: "x"}, strings.NewReader("x"))
		assert.Error(t, err)
	})
}
