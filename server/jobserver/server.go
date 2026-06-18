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
	"github.com/interline-io/log"
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
//	GET  /queues/{queue}/jobs/{jobId}/artifacts                      — list artifacts
//	GET  /queues/{queue}/jobs/{jobId}/artifacts/{artifactId}         — artifact metadata
//	GET  /queues/{queue}/jobs/{jobId}/artifacts/{artifactId}/download — download
func NewServer() (http.Handler, error) {
	r := chi.NewRouter()
	r.Post("/run", runJobRequest)
	r.Route("/queues/{queue}/jobs", func(r chi.Router) {
		r.Post("/", submitJobRequest)
		r.Get("/", listJobsRequest)
		r.Get("/{jobId}", statusJobRequest)
		r.Get("/{jobId}/watch", watchJobRequest)
		r.Post("/{jobId}/cancel", cancelJobRequest)
		r.Get("/{jobId}/artifacts", listArtifactsRequest)
		r.Get("/{jobId}/artifacts/{artifactId}", artifactMetaRequest)
		r.Get("/{jobId}/artifacts/{artifactId}/download", downloadArtifactRequest)
	})
	return r, nil
}

// runJobRequest runs synchronously via Runner — no queue, no tracking.
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
	if cfg.JobPolicy != nil {
		if err := cfg.JobPolicy.CanSubmit(req.Context(), job); err != nil {
			if errors.Is(err, jobs.ErrJobAccessDenied) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			internalError(w, req, "policy check failed", err)
			return
		}
	}
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
		if errors.Is(err, jobs.ErrJobAccessDenied) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		internalError(w, req, "submit failed", err)
		return
	}
	writeJSON(w, status)
}

const (
	queryUserID = "user_id"
	queryKind   = "kind"
	queryOffset = "offset"
	queryStates = "states"
	queryLimit  = "limit"
)

// maxListLimit caps client-supplied ?limit= as defense-in-depth.
const maxListLimit = 1000

func listJobsRequest(w http.ResponseWriter, req *http.Request) {
	sq, ok := requireStatusQueue(w, req)
	if !ok {
		return
	}
	q := req.URL.Query()
	opts := jobs.ListOptions{
		UserID: q.Get(queryUserID),
		Kind:   q.Get(queryKind),
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
		if n > maxListLimit {
			n = maxListLimit
		}
		opts.Limit = n
	}
	if v := q.Get(queryOffset); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			http.Error(w, "invalid offset", http.StatusBadRequest)
			return
		}
		opts.Offset = n
	}
	result, err := sq.List(req.Context(), opts)
	if err != nil {
		if errors.Is(err, jobs.ErrJobAccessDenied) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		internalError(w, req, "list failed", err)
		return
	}
	writeJSON(w, result)
}

func statusJobRequest(w http.ResponseWriter, req *http.Request) {
	sq, ok := requireStatusQueue(w, req)
	if !ok {
		return
	}
	st, err := sq.Status(req.Context(), chi.URLParam(req, "jobId"))
	if err != nil {
		mapJobLookupError(w, req, err)
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
		mapJobLookupError(w, req, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// watchJobRequest streams JobEvents as SSE until terminal (with "event: end")
// or client disconnect.
func watchJobRequest(w http.ResponseWriter, req *http.Request) {
	sq, ok := requireStatusQueue(w, req)
	if !ok {
		return
	}
	// ResponseController unwraps middleware-wrapped writers; bare type-assert
	// to http.Flusher breaks the moment anything wraps w.
	rc := http.NewResponseController(w)
	ctx := req.Context()
	ch, err := sq.Watch(ctx, chi.URLParam(req, "jobId"))
	if err != nil {
		mapJobLookupError(w, req, err)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
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
		if errors.Is(err, jobs.ErrUnknownQueue) {
			http.Error(w, "unknown queue", http.StatusNotFound)
			return nil, false
		}
		internalError(w, req, "queue lookup failed", err)
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

// scopeJobUserID stamps the authenticated user's ID onto the job. Admins
// may submit on behalf of any user, but default to themselves when the
// request doesn't name one — otherwise the job runs unattributed and workers
// fall back to the synthetic job user. Non-admins always get their own ID
// regardless of what the request body specified.
func scopeJobUserID(ctx context.Context, job *jobs.Job) {
	user := authn.ForContext(ctx)
	if user == nil {
		return
	}
	if user.HasRole("admin") {
		if job.Opts.UserID == "" {
			job.Opts.UserID = user.ID()
		}
		return
	}
	job.Opts.UserID = user.ID()
}

// mapJobLookupError collapses ErrJobNotFound and ErrJobAccessDenied to 404
// so authenticated-but-not-owner callers can't probe ID existence.
func mapJobLookupError(w http.ResponseWriter, req *http.Request, err error) {
	if errors.Is(err, jobs.ErrJobNotFound) || errors.Is(err, jobs.ErrJobAccessDenied) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	internalError(w, req, "job lookup failed", err)
}

// internalError logs err and returns a generic 500 so underlying details
// (DB messages, paths, etc.) don't leak to clients.
func internalError(w http.ResponseWriter, req *http.Request, msg string, err error) {
	log.For(req.Context()).Error().Err(err).Msg(msg)
	http.Error(w, "internal error", http.StatusInternalServerError)
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
