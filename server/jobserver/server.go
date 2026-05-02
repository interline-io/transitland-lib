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
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/jobs"
	"github.com/interline-io/transitland-lib/server/model"
)

// NewServer creates a simple api for submitting and running jobs.
func NewServer(queueName string, workers int) (http.Handler, error) {
	r := chi.NewRouter()
	r.HandleFunc("/add", addJobRequest)
	r.HandleFunc("/run", runJobRequest)
	r.Get("/jobs", listJobsRequest)
	r.Get("/jobs/{jobId}", statusJobRequest)
	r.Get("/jobs/{jobId}/watch", watchJobRequest)
	return r, nil
}

// addJobRequest enqueues the posted job. Responds with the resulting JobStatus.
func addJobRequest(w http.ResponseWriter, req *http.Request) {
	submitJobRequest(w, req, jobs.JobQueue.AddJob)
}

// runJobRequest runs the posted job synchronously. Responds with the terminal JobStatus.
func runJobRequest(w http.ResponseWriter, req *http.Request) {
	submitJobRequest(w, req, jobs.JobQueue.RunJob)
}

// submitJobRequest implements the shared shape of /add and /run: gate auth,
// parse the body, scope the user, invoke the queue method, and write back the
// JobStatus (or an HTTP error).
func submitJobRequest(w http.ResponseWriter, req *http.Request, do func(jobs.JobQueue, context.Context, jobs.Job) (jobs.JobStatus, error)) {
	queue := requireQueueAndAuth(w, req)
	if queue == nil {
		return
	}
	job, err := requestGetJob(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ctx := req.Context()
	scopeJobUserID(ctx, &job)
	status, err := do(queue, ctx, job)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, status)
}

// scopeJobUserID binds the job's UserID to the authenticated user. Admins may
// submit on behalf of any user (e.g. system jobs); non-admins always have
// their own ID stamped, regardless of what the request body specified.
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

// requireQueueAndAuth resolves the JobQueue and confirms the request is
// authenticated. On failure it writes the appropriate response and returns
// a nil queue; callers should return immediately in that case.
func requireQueueAndAuth(w http.ResponseWriter, req *http.Request) jobs.JobQueue {
	ctx := req.Context()
	queue := model.ForContext(ctx).JobQueue
	if queue == nil {
		http.Error(w, "no job queue available", http.StatusServiceUnavailable)
		return nil
	}
	if authn.ForContext(ctx) == nil {
		http.Error(w, "unauthenticated", http.StatusForbidden)
		return nil
	}
	return queue
}

// statusJobRequest returns the JobStatus for a single job.
// Returns 404 if the job is not visible to the caller (whether absent or not theirs).
func statusJobRequest(w http.ResponseWriter, req *http.Request) {
	queue := requireQueueAndAuth(w, req)
	if queue == nil {
		return
	}
	st, err := queue.Status(req.Context(), chi.URLParam(req, "jobId"))
	if err != nil {
		mapJobLookupError(w, err)
		return
	}
	writeJSON(w, st)
}

// listJobsRequest lists jobs visible to the caller. Non-admins see only their own.
// Query params: states (comma-separated), user_id, kind, limit, after.
func listJobsRequest(w http.ResponseWriter, req *http.Request) {
	queue := requireQueueAndAuth(w, req)
	if queue == nil {
		return
	}
	q := req.URL.Query()
	opts := jobs.JobListOptions{
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
	result, err := queue.ListJobs(req.Context(), opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, result)
}

// watchJobRequest streams JobEvents over Server-Sent Events. The stream closes
// after a terminal event (with a final "event: end" sentinel) or when the
// client disconnects.
func watchJobRequest(w http.ResponseWriter, req *http.Request) {
	queue := requireQueueAndAuth(w, req)
	if queue == nil {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	ctx := req.Context()
	ch, err := queue.Watch(ctx, chi.URLParam(req, "jobId"))
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
	// stream when the job sits in queued/running for a while with no transitions.
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

// mapJobLookupError translates queue lookup errors to HTTP. ErrJobAccessDenied
// from the queue means the caller is authenticated but not the owner; we return
// 404 to avoid leaking that the job ID exists.
func mapJobLookupError(w http.ResponseWriter, err error) {
	if errors.Is(err, jobs.ErrJobNotFound) || errors.Is(err, jobs.ErrJobAccessDenied) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// requestGetJob parses job from request body
func requestGetJob(req *http.Request) (jobs.Job, error) {
	var job jobs.Job
	err := json.NewDecoder(req.Body).Decode(&job)
	if err != nil {
		return job, errors.New("error parsing body")
	}
	return job, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
