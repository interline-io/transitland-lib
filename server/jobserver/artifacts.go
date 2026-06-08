package jobserver

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/interline-io/transitland-lib/server/model"
)

// artifactResponse is the JSON shape returned to clients. It deliberately omits
// the model's internal fields (storage_key, user_id) and adds a download URL
// that points back at this authenticated server (never a raw presigned URL).
type artifactResponse struct {
	ID          int       `json:"id"`
	JobID       string    `json:"job_id"`
	JobKind     string    `json:"job_kind"`
	Filename    string    `json:"filename"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	SHA1        string    `json:"sha1,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	DownloadURL string    `json:"download_url"`
}

func toArtifactResponse(a *model.JobArtifact, downloadURL string) artifactResponse {
	return artifactResponse{
		ID:          a.ID,
		JobID:       a.JobID,
		JobKind:     a.JobKind,
		Filename:    a.Filename,
		ContentType: a.ContentType,
		SizeBytes:   a.SizeBytes,
		SHA1:        a.SHA1,
		CreatedAt:   a.CreatedAt,
		DownloadURL: downloadURL,
	}
}

// requireArtifactReader returns the configured read side, or writes 501 if the
// active deployment has no artifact store wired.
func requireArtifactReader(w http.ResponseWriter, req *http.Request) (model.ArtifactReader, bool) {
	cfg := model.ForContext(req.Context())
	if cfg.ArtifactStoreFactory == nil {
		http.Error(w, "artifacts not configured", http.StatusNotImplemented)
		return nil, false
	}
	return cfg.ArtifactStoreFactory, true
}

// authorizeJob enforces the same access policy as the status endpoint: a
// successful sq.Status means the caller may read this job (and therefore its
// artifacts). Not-found and access-denied both collapse to 404.
func authorizeJob(w http.ResponseWriter, req *http.Request) (string, bool) {
	sq, ok := requireStatusQueue(w, req)
	if !ok {
		return "", false
	}
	jobID := chi.URLParam(req, "jobId")
	if _, err := sq.Status(req.Context(), jobID); err != nil {
		mapJobLookupError(w, req, err)
		return "", false
	}
	return jobID, true
}

// loadArtifact parses {artifactId}, loads the row, and verifies it belongs to
// jobID — preventing access to another job's artifact by guessing its id. The
// job itself has already been authorized by authorizeJob.
func loadArtifact(w http.ResponseWriter, req *http.Request, reader model.ArtifactReader, jobID string) (*model.JobArtifact, bool) {
	id, err := strconv.Atoi(chi.URLParam(req, "artifactId"))
	if err != nil || id <= 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return nil, false
	}
	art, err := reader.GetByID(req.Context(), id)
	if err != nil {
		if errors.Is(err, model.ErrArtifactNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
			return nil, false
		}
		internalError(w, req, "artifact lookup failed", err)
		return nil, false
	}
	if art.JobID != jobID {
		http.Error(w, "not found", http.StatusNotFound)
		return nil, false
	}
	return art, true
}

func listArtifactsRequest(w http.ResponseWriter, req *http.Request) {
	jobID, ok := authorizeJob(w, req)
	if !ok {
		return
	}
	reader, ok := requireArtifactReader(w, req)
	if !ok {
		return
	}
	arts, err := reader.ListByJob(req.Context(), jobID)
	if err != nil {
		internalError(w, req, "artifact list failed", err)
		return
	}
	// download_url is the collection path (this request) + /{id}/download.
	base := strings.TrimRight(req.URL.Path, "/")
	out := make([]artifactResponse, 0, len(arts))
	for _, a := range arts {
		out = append(out, toArtifactResponse(a, base+"/"+strconv.Itoa(a.ID)+"/download"))
	}
	writeJSON(w, map[string]any{"artifacts": out})
}

func artifactMetaRequest(w http.ResponseWriter, req *http.Request) {
	jobID, ok := authorizeJob(w, req)
	if !ok {
		return
	}
	reader, ok := requireArtifactReader(w, req)
	if !ok {
		return
	}
	art, ok := loadArtifact(w, req, reader, jobID)
	if !ok {
		return
	}
	// This request path is .../artifacts/{id}; download is one segment deeper.
	writeJSON(w, toArtifactResponse(art, strings.TrimRight(req.URL.Path, "/")+"/download"))
}

func downloadArtifactRequest(w http.ResponseWriter, req *http.Request) {
	jobID, ok := authorizeJob(w, req)
	if !ok {
		return
	}
	reader, ok := requireArtifactReader(w, req)
	if !ok {
		return
	}
	art, ok := loadArtifact(w, req, reader, jobID)
	if !ok {
		return
	}
	cfg := model.ForContext(req.Context())
	// serveArtifact writes its own success/error response and logs failures.
	serveArtifact(w, req, cfg.ArtifactStorage, art)
}
