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

// runJobRequest runs the posted job synchronously via Runner. No queue, no
// tracking; the response is a synthesized terminal JobStatus.
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

func listJobsRequest(w http.ResponseWriter, req *http.Request) {
	sq, ok := requireStatusQueue(w, req)
	if !ok {
		return
	}
	q := req.URL.Query()
	opts := jobs.ListOptions{
		UserID: q.Get("user_id"),
		Kind:   q.Get("kind"),
		After:  q.Get("after"),
	}
	if v := q.Get("states"); v != "" {
		for _, s := range strings.Split(v, ",") {
			if s = strings.TrimSpace(s); s != "" {
				opts.States = append(opts.States, jobs.JobState(s))
			}
		}
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		opts.Limit = n
	}
	result, err := sq.List(req.Context(), opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, result)
}

// statusJobRequest returns the JobStatus for a single job.
// Returns 404 if the job is not visible to the caller (whether absent or not theirs).
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

// watchJobRequest streams JobEvents over Server-Sent Events. The stream
// closes after a terminal event (with a final "event: end" sentinel) or
// when the client disconnects.
func watchJobRequest(w http.ResponseWriter, req *http.Request) {
	sq, ok := requireStatusQueue(w, req)
	if !ok {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
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
	flusher.Flush()
	// Heartbeat keeps intermediate proxies (nginx, ELBs) from closing an idle
	// stream when the job sits in queued/running for a while.
	heartbeat := time.NewTicker(sseHeartbeatInterval)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			fmt.Fprint(w, ": keepalive\n\n")
			flusher.Flush()
		case ev, open := <-ch:
			if !open {
				fmt.Fprint(w, "event: end\ndata: {}\n\n")
				flusher.Flush()
				return
			}
			b, err := json.Marshal(ev)
			if err != nil {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		}
	}
}

const sseHeartbeatInterval = 30 * time.Second

// requireQueue resolves the named queue and confirms auth. On failure it
// writes the response and returns ok=false.
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
	q := cfg.Jobs.Queue(chi.URLParam(req, "queue"))
	if q == nil {
		http.Error(w, "unknown queue", http.StatusNotFound)
		return nil, false
	}
	return q, true
}

// requireStatusQueue additionally type-asserts StatusQueue; the lifecycle
// endpoints can't run on queues that don't track jobs.
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
	job.UserID = user.ID()
}

// mapJobLookupError translates queue lookup errors to HTTP. ErrJobAccessDenied
// from the queue means the caller is authenticated but not the owner; we
// return 404 to avoid leaking that the job ID exists.
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
