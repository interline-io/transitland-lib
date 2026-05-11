package jobserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/jobs"
	"github.com/interline-io/transitland-lib/server/model"
)

// NewServer wires the job HTTP API. Routes:
//
//	POST /run                                    — runner-only synchronous (queue-less)
//	POST /queues/{queue}/jobs                    — submit
//	GET  /queues/{queue}/jobs                    — list
//	GET  /queues/{queue}/jobs/{jobId}            — status
//	GET  /queues/{queue}/jobs/{jobId}/watch      — SSE stream
//	POST /queues/{queue}/jobs/{jobId}/cancel     — cancel
func NewServer() (http.Handler, error) {
	r := chi.NewRouter()
	r.Post("/run", runJobRequest)
	r.Route("/queues/{queue}/jobs", func(r chi.Router) {
		r.Post("/", submitJobRequest)
		r.Get("/", listJobsRequest)
		r.Get("/{jobId}", statusJobRequest)
		r.Get("/{jobId}/watch", watchJobRequest)
		r.Post("/{jobId}/cancel", cancelJobRequest)
	})
	return r, nil
}

// runJobRequest runs the posted job synchronously via Runner — no queue, no
// tracking. The response is a synthesized terminal JobStatus.
func runJobRequest(w http.ResponseWriter, req *http.Request) {
	cfg := model.ForContext(req.Context())
	if cfg.JobRunner == nil {
		http.Error(w, "no job runner available", http.StatusServiceUnavailable)
		return
	}
	if authn.ForContext(req.Context()) == nil {
		http.Error(w, "unauthenticated", http.StatusForbidden)
		return
	}
	job, err := decodeJob(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	scopeJobUserID(req.Context(), &job)
	job.ID = uuid.NewString()
	startedAt := time.Now().UTC()
	runErr := cfg.JobRunner.Run(req.Context(), job)
	finishedAt := time.Now().UTC()
	state := jobs.JobStateSucceeded
	msg := ""
	if runErr != nil {
		state = jobs.JobStateFailed
		msg = runErr.Error()
	}
	writeJSON(w, jobs.JobStatus{
		State:       state,
		Job:         job,
		SubmittedAt: startedAt,
		StartedAt:   &startedAt,
		FinishedAt:  &finishedAt,
		Error:       msg,
	})
}

func submitJobRequest(w http.ResponseWriter, req *http.Request) {
	q, ok := requireQueue(w, req)
	if !ok {
		return
	}
	job, err := decodeJob(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	scopeJobUserID(req.Context(), &job)
	status, err := q.Submit(req.Context(), job)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, status)
}

// Query parameter names accepted by listJobsRequest.
const (
	queryUserID = "user_id"
	queryKind   = "kind"
	queryAfter  = "after"
	queryStates = "states"
	queryLimit  = "limit"
)

func listJobsRequest(w http.ResponseWriter, req *http.Request) {
	sq, ok := requireStatusQueue(w, req)
	if !ok {
		return
	}
	q := req.URL.Query()
	opts := jobs.ListOptions{
		UserID: q.Get(queryUserID),
		Kind:   q.Get(queryKind),
		After:  q.Get(queryAfter),
	}
	if v := q.Get(queryStates); v != "" {
		for _, s := range strings.Split(v, ",") {
			if s = strings.TrimSpace(s); s != "" {
				opts.States = append(opts.States, jobs.JobState(s))
			}
		}
	}
	if v := q.Get(queryLimit); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		opts.Limit = n
	}
	result, err := sq.List(req.Context(), opts)
	if err != nil {
		if errors.Is(err, jobs.ErrInvalidCursor) {
			http.Error(w, "invalid cursor", http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, result)
}

// statusJobRequest returns 404 when the job is not visible to the caller,
// regardless of whether it's absent or just not theirs.
func statusJobRequest(w http.ResponseWriter, req *http.Request) {
	sq, ok := requireStatusQueue(w, req)
	if !ok {
		return
	}
	st, err := sq.Status(req.Context(), chi.URLParam(req, "jobId"))
	if err != nil {
		mapJobLookupError(w, err)
		return
	}
	writeJSON(w, st)
}

func cancelJobRequest(w http.ResponseWriter, req *http.Request) {
	sq, ok := requireStatusQueue(w, req)
	if !ok {
		return
	}
	if err := sq.Cancel(req.Context(), chi.URLParam(req, "jobId")); err != nil {
		mapJobLookupError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// watchJobRequest streams JobEvents as SSE. Closes on terminal (with an
// "event: end" sentinel) or client disconnect.
func watchJobRequest(w http.ResponseWriter, req *http.Request) {
	sq, ok := requireStatusQueue(w, req)
	if !ok {
		return
	}
	// http.ResponseController navigates middleware-wrapped writers to find
	// Flusher; bare w.(http.Flusher) breaks the moment anything wraps w.
	rc := http.NewResponseController(w)
	ctx := req.Context()
	ch, err := sq.Watch(ctx, chi.URLParam(req, "jobId"))
	if err != nil {
		mapJobLookupError(w, err)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	// Flush fails when a middleware wraps without Unwrap; emit anyway so the
	// client at least sees the terminal event when the connection closes.
	_ = rc.Flush()
	// Heartbeat keeps proxies (nginx, ELBs) from closing the idle stream.
	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			_ = rc.Flush()
		case ev, open := <-ch:
			if !open {
				_, _ = fmt.Fprint(w, "event: end\ndata: {}\n\n")
				_ = rc.Flush()
				return
			}
			b, err := json.Marshal(ev)
			if err != nil {
				return
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
				return
			}
			_ = rc.Flush()
		}
	}
}

const sseHeartbeatInterval = 30 * time.Second

func requireQueue(w http.ResponseWriter, req *http.Request) (jobs.Queue, bool) {
	cfg := model.ForContext(req.Context())
	if cfg.Jobs == nil {
		http.Error(w, "no job backend available", http.StatusServiceUnavailable)
		return nil, false
	}
	if authn.ForContext(req.Context()) == nil {
		http.Error(w, "unauthenticated", http.StatusForbidden)
		return nil, false
	}
	q, err := cfg.Jobs.Queue(chi.URLParam(req, "queue"))
	if err != nil {
		http.Error(w, "unknown queue", http.StatusNotFound)
		return nil, false
	}
	return q, true
}

func requireStatusQueue(w http.ResponseWriter, req *http.Request) (jobs.StatusQueue, bool) {
	q, ok := requireQueue(w, req)
	if !ok {
		return nil, false
	}
	sq, ok := q.(jobs.StatusQueue)
	if !ok {
		http.Error(w, "queue does not support status", http.StatusNotImplemented)
		return nil, false
	}
	return sq, true
}

// scopeJobUserID binds the job's UserID to the authenticated user. Admins
// may submit on behalf of any user (e.g. system jobs); non-admins always
// have their own ID stamped, regardless of what the request body specified.
func scopeJobUserID(ctx context.Context, job *jobs.Job) {
	user := authn.ForContext(ctx)
	if user == nil {
		return
	}
	if user.HasRole("admin") {
		return
	}
	job.Opts.UserID = user.ID()
}

// mapJobLookupError returns 404 for both ErrJobNotFound and ErrJobAccessDenied
// so an authenticated-but-not-owner caller can't probe ID existence.
func mapJobLookupError(w http.ResponseWriter, err error) {
	if errors.Is(err, jobs.ErrJobNotFound) || errors.Is(err, jobs.ErrJobAccessDenied) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func decodeJob(req *http.Request) (jobs.Job, error) {
	var job jobs.Job
	if err := json.NewDecoder(req.Body).Decode(&job); err != nil {
		return job, errors.New("error parsing body")
	}
	return job, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
