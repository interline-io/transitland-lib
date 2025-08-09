package jobserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/interline-io/transitland-lib/internal/util"
	"github.com/interline-io/transitland-lib/model"
	"github.com/interline-io/transitland-mw/jobs"
)

// NewServer creates a simple api for submitting and running jobs.
func NewServer(queueName string, workers int) (http.Handler, error) {
	r := chi.NewRouter()
	r.HandleFunc("/add", addJobRequest)
	r.HandleFunc("/run", runJobRequest)
	return r, nil
}

// job response
type jobResponse struct {
	Status  string   `json:"status"`
	Success bool     `json:"success"`
	Error   string   `json:"error,omitempty"`
	Job     jobs.Job `json:"job"`
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
	} else if err := jobQueue.AddJob(ctx, job); err != nil {
		ret.Status = "failed"
		ret.Error = err.Error()
	} else {
		ret.Status = "added"
		ret.Success = true
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
	} else if err := jobQueue.RunJob(ctx, job); err != nil {
		ret.Status = "failed"
		ret.Error = err.Error()
	} else {
		ret.Status = "completed"
		ret.Success = true
	}
	writeJobResponse(ret, w)
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
