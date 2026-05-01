package jobserver

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/jobs"
	localjobs "github.com/interline-io/transitland-lib/server/jobs/local"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
)

type echoWorker struct {
	kind string
}

func (e *echoWorker) Kind() string                    { return e.kind }
func (e *echoWorker) Run(ctx context.Context) error   { return nil }

// testAuthMiddleware reads X-Test-User and X-Test-Roles headers and attaches
// the corresponding authn.User to the request context. Lets tests drive auth
// across an httptest.Server without a real auth backend.
func testAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Test-User")
		if id == "" {
			next.ServeHTTP(w, r)
			return
		}
		user := authn.NewCtxUser(id, "", "")
		if roles := r.Header.Get("X-Test-Roles"); roles != "" {
			for _, role := range strings.Split(roles, ",") {
				if role = strings.TrimSpace(role); role != "" {
					user = user.WithRoles(role)
				}
			}
		}
		next.ServeHTTP(w, r.WithContext(authn.WithUser(r.Context(), user)))
	})
}

func newTestServer(t *testing.T) (*httptest.Server, *localjobs.LocalJobs) {
	t.Helper()
	q := localjobs.NewLocalJobs()
	q.SetTerminalTTL(0) // disable eviction during tests
	if err := q.AddJobType(func() jobs.JobWorker { return &echoWorker{kind: "test"} }); err != nil {
		t.Fatal(err)
	}
	cfg := model.Config{JobQueue: q}
	h, err := NewServer("default", 1)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(testAuthMiddleware(model.AddConfig(cfg)(h)))
	t.Cleanup(srv.Close)
	return srv, q
}

func authedRequest(t *testing.T, method, url string, user authn.User) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if user != nil {
		req.Header.Set("X-Test-User", user.ID())
		if roles := user.Roles(); len(roles) > 0 {
			req.Header.Set("X-Test-Roles", strings.Join(roles, ","))
		}
	}
	return req
}

func TestStatusEndpoint(t *testing.T) {
	srv, q := newTestServer(t)
	owner := authn.NewCtxUser("alice", "", "")
	stranger := authn.NewCtxUser("bob", "", "")

	// Run a job as alice and grab its JobId.
	st, err := q.RunJob(authn.WithUser(context.Background(), owner), jobs.Job{JobType: "test", UserId: "alice"})
	if err != nil {
		t.Fatal(err)
	}

	// Owner: 200.
	resp, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/jobs/"+st.Job.JobId, owner))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var got jobs.JobStatus
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	assert.Equal(t, st.Job.JobId, got.Job.JobId)
	assert.Equal(t, jobs.JobStateSucceeded, got.State)

	// Stranger: 404 (no leak).
	resp, err = http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/jobs/"+st.Job.JobId, stranger))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Unknown id: 404.
	resp, err = http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/jobs/does-not-exist", owner))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Unauthenticated: 403.
	resp, err = http.Get(srv.URL + "/jobs/" + st.Job.JobId)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestListEndpoint(t *testing.T) {
	srv, q := newTestServer(t)
	owner := authn.NewCtxUser("alice", "", "")
	admin := authn.NewCtxUser("admin", "", "").WithRoles("admin")

	for i := 0; i < 3; i++ {
		_, err := q.RunJob(authn.WithUser(context.Background(), owner), jobs.Job{JobType: "test", UserId: "alice"})
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		_, err := q.RunJob(authn.WithUser(context.Background(), admin), jobs.Job{JobType: "test", UserId: "carol"})
		if err != nil {
			t.Fatal(err)
		}
	}

	// Admin sees all 5.
	resp, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/jobs?job_type=test", admin))
	if err != nil {
		t.Fatal(err)
	}
	var all jobs.JobListResult
	if err := json.NewDecoder(resp.Body).Decode(&all); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 5, len(all.Jobs))

	// Alice sees only her 3, even when asking for carol's.
	resp, err = http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/jobs?job_type=test&user_id=carol", owner))
	if err != nil {
		t.Fatal(err)
	}
	var mine jobs.JobListResult
	if err := json.NewDecoder(resp.Body).Decode(&mine); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	assert.Equal(t, 3, len(mine.Jobs))
	for _, st := range mine.Jobs {
		assert.Equal(t, "alice", st.Job.UserId)
	}

	// Comma-separated states filter.
	resp, err = http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/jobs?states=succeeded,failed", admin))
	if err != nil {
		t.Fatal(err)
	}
	var byState jobs.JobListResult
	if err := json.NewDecoder(resp.Body).Decode(&byState); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	assert.Equal(t, 5, len(byState.Jobs))

	// Unauthenticated: 403.
	resp, err = http.Get(srv.URL + "/jobs")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestWatchEndpointTerminalReplay(t *testing.T) {
	srv, q := newTestServer(t)
	owner := authn.NewCtxUser("alice", "", "")

	st, err := q.RunJob(authn.WithUser(context.Background(), owner), jobs.Job{JobType: "test", UserId: "alice"})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/jobs/"+st.Job.JobId+"/watch", owner))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	events, sawEnd := readSSE(t, resp.Body, 2*time.Second)
	if assert.NotEmpty(t, events) {
		assert.True(t, events[len(events)-1].State.Terminal())
	}
	assert.True(t, sawEnd, "expected a final event: end sentinel")
}

func TestWatchEndpointStreaming(t *testing.T) {
	srv, q := newTestServer(t)
	owner := authn.NewCtxUser("alice", "", "")
	q.AddQueue("default", 1)

	// Submit but don't run yet — job stays in queued state.
	st, err := q.AddJob(authn.WithUser(context.Background(), owner), jobs.Job{JobType: "test", UserId: "alice"})
	if err != nil {
		t.Fatal(err)
	}

	// Open the watch before the queue starts running.
	resp, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/jobs/"+st.Job.JobId+"/watch", owner))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Run the queue; the worker pool will pick up the queued job and drive it
	// to terminal, transitioning the registry entry the watcher is attached to.
	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()
	go func() {
		time.Sleep(500 * time.Millisecond)
		_ = q.Stop(runCtx)
	}()
	go func() { _ = q.Run(runCtx) }()

	events, sawEnd := readSSE(t, resp.Body, 3*time.Second)
	assert.True(t, sawEnd)
	if assert.NotEmpty(t, events) {
		assert.True(t, events[len(events)-1].State.Terminal())
	}
}

// readSSE reads an SSE stream until end sentinel or timeout. Returns parsed
// JobEvent payloads (excluding the end sentinel) and whether the end sentinel
// was observed.
func readSSE(t *testing.T, body interface {
	Read(p []byte) (n int, err error)
}, timeout time.Duration) ([]jobs.JobEvent, bool) {
	t.Helper()
	type result struct {
		events []jobs.JobEvent
		sawEnd bool
	}
	done := make(chan result, 1)
	go func() {
		var events []jobs.JobEvent
		sawEnd := false
		scanner := bufio.NewScanner(body)
		var eventName string
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case strings.HasPrefix(line, "event: "):
				eventName = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				data := strings.TrimPrefix(line, "data: ")
				if eventName == "end" {
					sawEnd = true
					done <- result{events, sawEnd}
					return
				}
				var ev jobs.JobEvent
				if err := json.Unmarshal([]byte(data), &ev); err == nil {
					events = append(events, ev)
				}
			case line == "":
				eventName = ""
			}
		}
		done <- result{events, sawEnd}
	}()
	select {
	case r := <-done:
		return r.events, r.sawEnd
	case <-time.After(timeout):
		t.Fatal("SSE read timed out")
		return nil, false
	}
}
