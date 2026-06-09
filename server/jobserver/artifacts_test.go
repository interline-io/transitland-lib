package jobserver

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/finders/artifactstore"
	"github.com/interline-io/transitland-lib/server/jobs"
	localjobs "github.com/interline-io/transitland-lib/server/jobs/local"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
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
		// Simulate a path-rewriting ingress: download_url must include this
		// public prefix, not just the path this server sees.
		RestPrefix: "https://example.test/api",
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

	// An owned artifact with its bytes on the local store.
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
	get := func(t *testing.T, url string, user authn.User) *http.Response {
		t.Helper()
		resp, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, url, user))
		require.NoError(t, err)
		return resp
	}

	t.Run("list (owner) returns the artifact without internal fields", func(t *testing.T) {
		resp := get(t, base, owner)
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
		assert.NotContains(t, a0, "storage_key")
		assert.NotContains(t, a0, "user_id")
		// download_url carries the RestPrefix (correct behind a path-rewriting
		// ingress) and points at this artifact's download route.
		dlURL, _ := a0["download_url"].(string)
		assert.True(t, strings.HasPrefix(dlURL, "https://example.test/api/"), "download_url should include RestPrefix: %q", dlURL)
		assert.True(t, strings.HasSuffix(dlURL, "/artifacts/1/download"), "download_url: %q", dlURL)
	})

	t.Run("list (stranger) is 404", func(t *testing.T) {
		resp := get(t, base, stranger)
		resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("download (owner) streams bytes and headers", func(t *testing.T) {
		resp := get(t, base+"/1/download", owner)
		dl, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, content, dl)
		assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))
		assert.Contains(t, resp.Header.Get("Content-Disposition"), "out.txt")
	})

	t.Run("another job's artifact is 404 (IDOR guard)", func(t *testing.T) {
		resp := get(t, base+"/2", owner)
		resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("unknown artifact id is 404", func(t *testing.T) {
		resp := get(t, base+"/999", owner)
		resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("unauthenticated is 403", func(t *testing.T) {
		resp := get(t, base, nil)
		resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
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

func TestArtifactsStorageNotConfigured(t *testing.T) {
	// A factory is wired but ArtifactStorage is empty. The read side could list
	// rows, but bytes can't be served, so the deployment is treated as
	// not-configured: a consistent 501, not a 200 list followed by a 500 on the
	// download path from request.GetStore("").
	owner := authn.NewCtxUser("alice", "", "")
	factory := &fakeArtifactFactory{byID: map[int]*model.JobArtifact{}}
	srv, backend := newArtifactTestServer(t, factory, "")
	st := runOneAndStop(t, backend, authn.WithUser(context.Background(), owner),
		jobs.Job{Kind: "test", Opts: jobs.JobOpts{UserID: "alice"}})
	url := srv.URL + "/queues/" + testQueue + "/jobs/" + st.Job.ID + "/artifacts"
	resp, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, url, owner))
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotImplemented, resp.StatusCode)
}

// artifactWriter is a worker that publishes one artifact through the per-job
// cfg.Artifacts handle — the worker-facing half of the feature.
type artifactWriter struct {
	filename, content string
}

func (w *artifactWriter) Kind() string { return "artifact-writer" }
func (w *artifactWriter) Run(ctx context.Context) error {
	store := model.ForContext(ctx).Artifacts
	if store == nil {
		return errors.New("no artifact store in worker context")
	}
	_, err := store.CreateReader(ctx, model.ArtifactOpts{Filename: w.filename, ContentType: "text/plain"}, strings.NewReader(w.content))
	return err
}

// jobConfigMiddleware installs cfg plus the per-job artifact handle into the
// worker context, standing in for the config + artifacts job middleware that
// lives in the tlv2 deployment layer.
func jobConfigMiddleware(cfg model.Config, factory model.ArtifactStoreFactory) jobs.Middleware {
	return func(inner jobs.Worker, j jobs.Job) jobs.Worker {
		return &configWorker{Worker: inner, cfg: cfg, factory: factory, job: j}
	}
}

type configWorker struct {
	jobs.Worker
	cfg     model.Config
	factory model.ArtifactStoreFactory
	job     jobs.Job
}

func (w *configWorker) Run(ctx context.Context) error {
	cfg := w.cfg
	cfg.Artifacts = w.factory.For(w.job.ID, w.job.Opts.UserID, w.job.Kind)
	return w.Worker.Run(model.WithConfig(ctx, cfg))
}

// TestArtifactEndToEnd runs the whole vertical slice against a real store:
// submit a job on the local queue, let the worker write an artifact, then list
// and download it back through the jobserver HTTP routes. Requires a Postgres
// test DB; skips otherwise.
func TestArtifactEndToEnd(t *testing.T) {
	if msg, ok := testutil.CheckTestDB(); !ok {
		t.Skip(msg)
	}
	db := testutil.MustOpenTestDB(t)
	dir := t.TempDir()
	owner := authn.NewCtxUser("alice", "", "")
	factory := artifactstore.NewStore(db, dir)

	const content = "end-to-end artifact bytes"

	runner := jobs.NewRunner()
	backend := localjobs.NewLocalBackend(runner, map[string]localjobs.QueueOpts{testQueue: {Workers: 1}}, nil)
	require.NoError(t, runner.Register(func() jobs.Worker {
		return &artifactWriter{filename: "report.txt", content: content}
	}))
	cfg := model.Config{
		Jobs:                 backend,
		JobRunner:            runner,
		Checker:              &authz.AllowAllChecker{},
		ArtifactStoreFactory: factory,
		ArtifactStorage:      dir,
	}
	runner.Use(jobConfigMiddleware(cfg, factory))

	h, err := NewServer()
	require.NoError(t, err)
	srv := httptest.NewServer(testAuthMiddleware(model.AddConfig(cfg)(h)))
	t.Cleanup(srv.Close)

	st := runOneAndStop(t, backend, authn.WithUser(context.Background(), owner),
		jobs.Job{Kind: "artifact-writer", Opts: jobs.JobOpts{UserID: "alice"}})
	require.Equal(t, jobs.JobStateSucceeded, st.State, "worker error: %s", st.Error)
	jobID := st.Job.ID
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), "DELETE FROM tl_job_artifacts WHERE job_id = $1", jobID)
	})

	base := srv.URL + "/queues/" + testQueue + "/jobs/" + jobID + "/artifacts"

	// List: the artifact the worker wrote is now visible to the owner.
	resp, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, base, owner))
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var listResp struct {
		Artifacts []struct {
			ID        int    `json:"id"`
			Filename  string `json:"filename"`
			SizeBytes int64  `json:"size_bytes"`
		} `json:"artifacts"`
	}
	require.NoError(t, json.Unmarshal(body, &listResp))
	require.Len(t, listResp.Artifacts, 1)
	got := listResp.Artifacts[0]
	assert.Equal(t, "report.txt", got.Filename)
	assert.Equal(t, int64(len(content)), got.SizeBytes)

	// Download: the exact bytes the worker wrote stream back.
	resp, err = http.DefaultClient.Do(authedRequest(t, http.MethodGet, base+"/"+strconv.Itoa(got.ID)+"/download", owner))
	require.NoError(t, err)
	dl, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, content, string(dl))
	assert.Contains(t, resp.Header.Get("Content-Disposition"), "report.txt")
}
