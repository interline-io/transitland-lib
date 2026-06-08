package jobserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/jobs"
	localjobs "github.com/interline-io/transitland-lib/server/jobs/local"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeArtifactFactory is an in-memory model.ArtifactStoreFactory for handler
// tests (no DB). The read methods are all that the jobserver exercises.
type fakeArtifactFactory struct {
	byID map[int]*model.JobArtifact
}

func (f *fakeArtifactFactory) For(jobID, userID, kind string) model.ArtifactStore { return nil }

func (f *fakeArtifactFactory) ListByJob(ctx context.Context, jobID string) ([]*model.JobArtifact, error) {
	var out []*model.JobArtifact
	for _, a := range f.byID {
		if a.JobID == jobID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (f *fakeArtifactFactory) GetByID(ctx context.Context, id int) (*model.JobArtifact, error) {
	if a, ok := f.byID[id]; ok {
		return a, nil
	}
	return nil, model.ErrArtifactNotFound
}

func newArtifactTestServer(t *testing.T, factory model.ArtifactStoreFactory, storage string) (*httptest.Server, *localjobs.LocalBackend) {
	t.Helper()
	runner := jobs.NewRunner()
	backend := localjobs.NewLocalBackend(runner, map[string]localjobs.QueueOpts{testQueue: {Workers: 1}}, nil)
	if err := runner.Register(func() jobs.Worker { return &echoWorker{kind: "test"} }); err != nil {
		t.Fatal(err)
	}
	cfg := model.Config{
		Jobs:                 backend,
		JobRunner:            runner,
		Checker:              &authz.AllowAllChecker{},
		ArtifactStoreFactory: factory,
		ArtifactStorage:      storage,
	}
	h, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(testAuthMiddleware(model.AddConfig(cfg)(h)))
	t.Cleanup(srv.Close)
	return srv, backend
}

func TestArtifactEndpoints(t *testing.T) {
	owner := authn.NewCtxUser("alice", "", "")
	stranger := authn.NewCtxUser("bob", "", "")
	dir := t.TempDir()

	factory := &fakeArtifactFactory{byID: map[int]*model.JobArtifact{}}
	srv, backend := newArtifactTestServer(t, factory, dir)

	// A real, owned, terminal job so sq.Status authorizes alice.
	st := runOneAndStop(t, backend, authn.WithUser(context.Background(), owner),
		jobs.Job{Kind: "test", Opts: jobs.JobOpts{UserID: "alice"}})
	jobID := st.Job.ID

	// A downloadable artifact for that job, with its bytes on the local store.
	content := []byte("artifact-bytes")
	key := filepath.ToSlash(filepath.Join("job-artifacts", jobID, "abc123", "out.txt"))
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dir, key)), 0o777))
	require.NoError(t, os.WriteFile(filepath.Join(dir, key), content, 0o666))
	art := &model.JobArtifact{
		JobID: jobID, JobKind: "test", UserID: "alice",
		Filename: "out.txt", ContentType: "text/plain",
		SizeBytes: int64(len(content)), SHA1: "deadbeef", StorageKey: key,
	}
	art.ID = 1
	factory.byID[1] = art

	// An artifact owned by a DIFFERENT job, for the IDOR cross-check.
	other := &model.JobArtifact{JobID: "some-other-job", Filename: "secret.txt", StorageKey: "k", SizeBytes: 1}
	other.ID = 2
	factory.byID[2] = other

	base := srv.URL + "/queues/" + testQueue + "/jobs/" + jobID + "/artifacts"

	// List (owner): 200 + one artifact; internal fields never leak.
	resp, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, base, owner))
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var listResp struct {
		Artifacts []map[string]any `json:"artifacts"`
	}
	require.NoError(t, json.Unmarshal(body, &listResp))
	require.Len(t, listResp.Artifacts, 1)
	a0 := listResp.Artifacts[0]
	assert.Equal(t, "out.txt", a0["filename"])
	assert.Contains(t, a0, "download_url")
	assert.NotContains(t, a0, "storage_key")
	assert.NotContains(t, a0, "user_id")

	// List (stranger): 404 — does not own the job.
	resp, err = http.DefaultClient.Do(authedRequest(t, http.MethodGet, base, stranger))
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Download (owner): 200 + bytes + download headers (Local stream branch).
	resp, err = http.DefaultClient.Do(authedRequest(t, http.MethodGet, base+"/1/download", owner))
	require.NoError(t, err)
	dl, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, content, dl)
	assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))
	assert.Contains(t, resp.Header.Get("Content-Disposition"), "out.txt")

	// Metadata for an artifact belonging to another job: 404 (IDOR guard).
	resp, err = http.DefaultClient.Do(authedRequest(t, http.MethodGet, base+"/2", owner))
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Unknown artifact id: 404.
	resp, err = http.DefaultClient.Do(authedRequest(t, http.MethodGet, base+"/999", owner))
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Unauthenticated: 403.
	resp, err = http.Get(base)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestArtifactsNotConfigured(t *testing.T) {
	owner := authn.NewCtxUser("alice", "", "")
	srv, backend := newArtifactTestServer(t, nil, "")
	st := runOneAndStop(t, backend, authn.WithUser(context.Background(), owner),
		jobs.Job{Kind: "test", Opts: jobs.JobOpts{UserID: "alice"}})
	url := srv.URL + "/queues/" + testQueue + "/jobs/" + st.Job.ID + "/artifacts"
	resp, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, url, owner))
	require.NoError(t, err)
	resp.Body.Close()
	// Owns the job (authorized) but no artifact store is wired -> 501.
	assert.Equal(t, http.StatusNotImplemented, resp.StatusCode)
}
