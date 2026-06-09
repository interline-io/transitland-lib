package jobserver

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/server/auth/authn"
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
// active deployment has no artifact store wired. A factory with no storage URL
// counts as not-wired: the read side could list rows, but the download path
// can't fetch bytes (request.GetStore("") fails). Gating both halves here keeps
// all three endpoints on one consistent 501 instead of leaking a 500 from the
// empty GetStore deeper in downloadArtifactRequest.
func requireArtifactReader(w http.ResponseWriter, req *http.Request) (model.ArtifactReader, bool) {
	cfg := model.ForContext(req.Context())
	if cfg.ArtifactStoreFactory == nil || cfg.ArtifactStorage == "" {
		http.Error(w, "artifacts not configured", http.StatusNotImplemented)
		return nil, false
	}
	return cfg.ArtifactStoreFactory, true
}

// artifactPrecheck validates the queue path and authentication shared by every
// artifact endpoint and returns the {jobId} segment. It deliberately does NOT
// consult job status: artifact authorization is by the artifact row's owner (see
// authorizeArtifactOwner), so artifacts stay reachable after the job is pruned
// from its backend (river retention, Argo GC, a local restart). Queues are
// static config and outlive individual jobs, so requireQueue still holds.
func artifactPrecheck(w http.ResponseWriter, req *http.Request) (string, bool) {
	if _, ok := requireQueue(w, req); !ok {
		return "", false
	}
	return chi.URLParam(req, "jobId"), true
}

// authorizeArtifactOwner reports whether the caller may read an artifact owned
// by ownerUserID. It mirrors jobs.CreatorOrAdmin.CanRead — admins read any,
// otherwise the caller must be the submitter — but evaluates it against the
// durable artifact row (user_id, copied from Job.Opts.UserID at create time)
// rather than live job status, so access outlives the job.
func authorizeArtifactOwner(req *http.Request, ownerUserID string) bool {
	user := authn.ForContext(req.Context())
	if user == nil {
		return false
	}
	if user.HasRole("admin") {
		return true
	}
	return user.ID() != "" && user.ID() == ownerUserID
}

// loadArtifact parses {artifactId}, loads the row, verifies it belongs to jobID
// (so a guessed id from another job is rejected), and authorizes the caller as
// the owner or an admin. Not-found, wrong-job, and not-owned all collapse to 404
// so the boundary can't be probed. Authorization is against the row's user_id,
// not live job status, so it survives the job being pruned.
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
	if art.JobID != jobID || !authorizeArtifactOwner(req, art.UserID) {
		http.Error(w, "not found", http.StatusNotFound)
		return nil, false
	}
	return art, true
}

func listArtifactsRequest(w http.ResponseWriter, req *http.Request) {
	jobID, ok := artifactPrecheck(w, req)
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
	// Prefix with RestPrefix (the ingress-stripped prefix) so download_url is
	// correct behind a path-rewriting ingress, as the REST pagination links do.
	cfg := model.ForContext(req.Context())
	base := cfg.RestPrefix + strings.TrimRight(req.URL.Path, "/")
	out := make([]artifactResponse, 0, len(arts))
	for _, a := range arts {
		// A job's artifacts all share one submitter, so a non-owner sees an empty
		// list rather than a 404 — revealing nothing about whether the job existed.
		if !authorizeArtifactOwner(req, a.UserID) {
			continue
		}
		out = append(out, toArtifactResponse(a, base+"/"+strconv.Itoa(a.ID)+"/download"))
	}
	writeJSON(w, map[string]any{"artifacts": out})
}

func artifactMetaRequest(w http.ResponseWriter, req *http.Request) {
	jobID, ok := artifactPrecheck(w, req)
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
	// Prefix with RestPrefix so the link is correct behind a path-rewriting ingress.
	cfg := model.ForContext(req.Context())
	writeJSON(w, toArtifactResponse(art, cfg.RestPrefix+strings.TrimRight(req.URL.Path, "/")+"/download"))
}

func downloadArtifactRequest(w http.ResponseWriter, req *http.Request) {
	jobID, ok := artifactPrecheck(w, req)
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
	store, err := request.GetStore(cfg.ArtifactStorage)
	if err != nil {
		internalError(w, req, "artifact storage unavailable", err)
		return
	}
	// serveArtifact writes its own success/error response and logs failures.
	serveArtifact(w, req, store, art)
}
