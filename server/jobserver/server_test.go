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

const testQueue = "default"

type echoWorker struct {
	kind string
}

func (e *echoWorker) Kind() string                  { return e.kind }
func (e *echoWorker) Run(ctx context.Context) error { return nil }

// testAuthMiddleware reads X-Test-User and X-Test-Roles headers and attaches
// the corresponding authn.User to the request context.
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

func newTestServer(t *testing.T) (*httptest.Server, *localjobs.LocalBackend, *jobs.Runner) {
	t.Helper()
	runner := jobs.NewRunner()
	backend := localjobs.NewLocalBackend(runner, map[string]localjobs.QueueOpts{
		testQueue: {Workers: 1, TerminalTTL: -1}, // disable eviction during tests
	})
	if err := runner.Register(func() jobs.Worker { return &echoWorker{kind: "test"} }); err != nil {
		t.Fatal(err)
	}
	cfg := model.Config{Jobs: backend, JobRunner: runner}
	h, err := NewServer()
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(testAuthMiddleware(model.AddConfig(cfg)(h)))
	t.Cleanup(srv.Close)
	return srv, backend, runner
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

// runOneAndStop submits a single job, runs the backend until that job
// reaches a terminal state, and stops it. Returns the terminal status.
func runOneAndStop(t *testing.T, backend *localjobs.LocalBackend, ctx context.Context, job jobs.Job) jobs.JobStatus {
	t.Helper()
	q := backend.Queue(testQueue)
	st, err := q.Submit(ctx, job)
	if err != nil {
		t.Fatal(err)
	}
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = backend.Run(runCtx) }()
	sq := q.(jobs.StatusQueue)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		got, _ := sq.Status(ctx, st.Job.ID)
		if got.State.Terminal() {
			_ = backend.Stop(runCtx)
			return got
		}
		time.Sleep(10 * time.Millisecond)
	}
	_ = backend.Stop(runCtx)
	t.Fatal("job did not reach terminal state")
	return jobs.JobStatus{}
}

func TestStatusEndpoint(t *testing.T) {
	srv, backend, _ := newTestServer(t)
	owner := authn.NewCtxUser("alice", "", "")
	stranger := authn.NewCtxUser("bob", "", "")

	st := runOneAndStop(t, backend, authn.WithUser(context.Background(), owner), jobs.Job{Kind: "test", UserID: "alice"})

	url := srv.URL + "/queues/" + testQueue + "/jobs/" + st.Job.ID

	// Owner: 200.
	resp, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, url, owner))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var got jobs.JobStatus
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	assert.Equal(t, st.Job.ID, got.Job.ID)
	assert.Equal(t, jobs.JobStateSucceeded, got.State)

	// Stranger: 404 (no leak).
	resp, err = http.DefaultClient.Do(authedRequest(t, http.MethodGet, url, stranger))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Unknown id: 404.
	resp, err = http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/queues/"+testQueue+"/jobs/does-not-exist", owner))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Unknown queue: 404.
	resp, err = http.DefaultClient.Do(authedRequest(t, http.MethodGet, srv.URL+"/queues/nope/jobs/"+st.Job.ID, owner))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	// Unauthenticated: 403.
	resp, err = http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestListEndpoint(t *testing.T) {
	srv, backend, _ := newTestServer(t)
	owner := authn.NewCtxUser("alice", "", "")
	admin := authn.NewCtxUser("admin", "", "").WithRoles("admin")

	q := backend.Queue(testQueue)
	for i := 0; i < 3; i++ {
		_, err := q.Submit(authn.WithUser(context.Background(), owner), jobs.Job{Kind: "test", UserID: "alice"})
		if err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 2; i++ {
		_, err := q.Submit(authn.WithUser(context.Background(), admin), jobs.Job{Kind: "test", UserID: "carol"})
		if err != nil {
			t.Fatal(err)
		}
	}
	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = backend.Run(runCtx) }()
	sq := q.(jobs.StatusQueue)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		res, _ := sq.List(authn.WithUser(context.Background(), admin), jobs.ListOptions{})
		terminal := 0
		for _, st := range res.Jobs {
			if st.State.Terminal() {
				terminal++
			}
		}
		if terminal == 5 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	_ = backend.Stop(runCtx)

	listURL := srv.URL + "/queues/" + testQueue + "/jobs"

	// Admin sees all 5.
	resp, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, listURL+"?kind=test", admin))
	if err != nil {
		t.Fatal(err)
	}
	var all jobs.ListResult
	if err := json.NewDecoder(resp.Body).Decode(&all); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 5, len(all.Jobs))

	// Alice sees only her 3 even when asking for carol's.
	resp, err = http.DefaultClient.Do(authedRequest(t, http.MethodGet, listURL+"?kind=test&user_id=carol", owner))
	if err != nil {
		t.Fatal(err)
	}
	var mine jobs.ListResult
	if err := json.NewDecoder(resp.Body).Decode(&mine); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	assert.Equal(t, 3, len(mine.Jobs))
	for _, st := range mine.Jobs {
		assert.Equal(t, "alice", st.Job.UserID)
	}

	// Comma-separated states filter.
	resp, err = http.DefaultClient.Do(authedRequest(t, http.MethodGet, listURL+"?states=succeeded,failed", admin))
	if err != nil {
		t.Fatal(err)
	}
	var byState jobs.ListResult
	if err := json.NewDecoder(resp.Body).Decode(&byState); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	assert.Equal(t, 5, len(byState.Jobs))

	// Unauthenticated: 403.
	resp, err = http.Get(listURL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestWatchEndpointTerminalReplay(t *testing.T) {
	srv, backend, _ := newTestServer(t)
	owner := authn.NewCtxUser("alice", "", "")

	st := runOneAndStop(t, backend, authn.WithUser(context.Background(), owner), jobs.Job{Kind: "test", UserID: "alice"})

	url := srv.URL + "/queues/" + testQueue + "/jobs/" + st.Job.ID + "/watch"
	resp, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, url, owner))
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
	srv, backend, _ := newTestServer(t)
	owner := authn.NewCtxUser("alice", "", "")

	q := backend.Queue(testQueue)
	st, err := q.Submit(authn.WithUser(context.Background(), owner), jobs.Job{Kind: "test", UserID: "alice"})
	if err != nil {
		t.Fatal(err)
	}

	url := srv.URL + "/queues/" + testQueue + "/jobs/" + st.Job.ID + "/watch"
	resp, err := http.DefaultClient.Do(authedRequest(t, http.MethodGet, url, owner))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	runCtx, cancelRun := context.WithCancel(context.Background())
	defer cancelRun()
	go func() {
		time.Sleep(500 * time.Millisecond)
		_ = backend.Stop(runCtx)
	}()
	go func() { _ = backend.Run(runCtx) }()

	events, sawEnd := readSSE(t, resp.Body, 3*time.Second)
	assert.True(t, sawEnd)
	if assert.NotEmpty(t, events) {
		assert.True(t, events[len(events)-1].State.Terminal())
	}
}

// readSSE reads an SSE stream until end sentinel or timeout. Returns parsed
// JobEvent payloads (excluding the end sentinel) and whether the end
// sentinel was observed.
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
