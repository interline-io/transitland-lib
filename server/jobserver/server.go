package jobserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/interline-io/transitland-lib/internal/util"
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

// job response
type jobResponse struct {
	Status    string          `json:"status"`
	Success   bool            `json:"success"`
	Error     string          `json:"error,omitempty"`
	Job       jobs.Job        `json:"job"`
	JobStatus *jobs.JobStatus `json:"job_status,omitempty"`
}

// addJobRequest adds the request to the appropriate queue
func addJobRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	job, err := requestGetJob(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// add job to queue
	ret := jobResponse{
		Job: job,
	}
	if jobQueue := model.ForContext(ctx).JobQueue; jobQueue == nil {
		ret.Status = "failed"
		ret.Error = "no job queue available"
	} else if status, err := jobQueue.AddJob(ctx, job); err != nil {
		ret.Status = "failed"
		ret.Error = err.Error()
	} else {
		ret.Status = "added"
		ret.Success = true
		ret.Job = status.Job
		ret.JobStatus = &status
	}
	writeJobResponse(ret, w)
}

// runJobRequest runs the job directly
func runJobRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	job, err := requestGetJob(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// run job directly
	ret := jobResponse{
		Job: job,
	}
	if jobQueue := model.ForContext(ctx).JobQueue; jobQueue == nil {
		ret.Status = "failed"
		ret.Error = "no job queue available"
	} else if status, err := jobQueue.RunJob(ctx, job); err != nil {
		ret.Status = "failed"
		ret.Error = err.Error()
	} else {
		ret.Status = "completed"
		ret.Success = true
		ret.Job = status.Job
		ret.JobStatus = &status
	}
	writeJobResponse(ret, w)
}

// statusJobRequest returns the JobStatus for a single job.
// Returns 404 if the job is not visible to the caller (whether absent or not theirs).
func statusJobRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	queue := model.ForContext(ctx).JobQueue
	if queue == nil {
		http.Error(w, "no job queue available", http.StatusServiceUnavailable)
		return
	}
	if authn.ForContext(ctx) == nil {
		http.Error(w, "unauthenticated", http.StatusForbidden)
		return
	}
	jobId := chi.URLParam(req, "jobId")
	st, err := queue.Status(ctx, jobId)
	if err != nil {
		mapJobLookupError(w, err)
		return
	}
	writeJSON(w, st)
}

// listJobsRequest lists jobs visible to the caller. Non-admins see only their own.
// Query params: states (comma-separated), user_id, job_type, limit, after.
func listJobsRequest(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	queue := model.ForContext(ctx).JobQueue
	if queue == nil {
		http.Error(w, "no job queue available", http.StatusServiceUnavailable)
		return
	}
	if authn.ForContext(ctx) == nil {
		http.Error(w, "unauthenticated", http.StatusForbidden)
		return
	}
	q := req.URL.Query()
	opts := jobs.JobListOptions{
		UserId:  q.Get("user_id"),
		JobType: q.Get("job_type"),
		After:   q.Get("after"),
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
	result, err := queue.ListJobs(ctx, opts)
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
	ctx := req.Context()
	queue := model.ForContext(ctx).JobQueue
	if queue == nil {
		http.Error(w, "no job queue available", http.StatusServiceUnavailable)
		return
	}
	if authn.ForContext(ctx) == nil {
		http.Error(w, "unauthenticated", http.StatusForbidden)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	jobId := chi.URLParam(req, "jobId")
	ch, err := queue.Watch(ctx, jobId)
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
// 404 to avoid leaking that the JobId exists.
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

// writeJobResponse writes job response
func writeJobResponse(ret jobResponse, w http.ResponseWriter) {
	if rj, err := json.Marshal(ret); err != nil {
		util.WriteJsonError(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	} else {
		w.Write(rj)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
