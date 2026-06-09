package model

import (
	"context"
	"errors"
	"io"

	"github.com/interline-io/transitland-lib/tt"
)

// ErrArtifactNotFound is returned by ArtifactReader.GetByID when no row matches.
var ErrArtifactNotFound = errors.New("artifact not found")

// JobArtifact is a file produced by a background job, for the submitting user to
// download. It is both the tl_job_artifacts row and the artifact-API DTO.
//
// JobID is opaque: a river_job id, local uuid, or Argo workflow name, never a
// foreign key. UserID and StorageKey are internal (json:"-") and must not reach
// API clients.
type JobArtifact struct {
	JobID       string `db:"job_id" json:"job_id"`
	JobKind     string `db:"job_kind" json:"job_kind"`
	UserID      string `db:"user_id" json:"-"`
	Filename    string `db:"filename" json:"filename"`
	ContentType string `db:"content_type" json:"content_type"`
	SizeBytes   int64  `db:"size_bytes" json:"size_bytes"`
	SHA1        string `db:"sha1" json:"sha1,omitempty"`
	StorageKey  string `db:"storage_key" json:"-"`
	tt.DatabaseEntity
	tt.Timestamps
}

// TableName implements the tldb table-name interface.
func (e *JobArtifact) TableName() string {
	return "tl_job_artifacts"
}

// ArtifactOpts are the caller-supplied fields when a worker publishes an
// artifact. JobID, StorageKey, SizeBytes and SHA1 are filled in by the store.
type ArtifactOpts struct {
	Filename    string
	ContentType string
}

// ArtifactStore is the per-job handle a worker uses to publish files. A scoped
// instance (bound to the executing job's id/user/kind) is resolved by
// JobArtifacts(ctx) from the job's JobMeta and Config.ArtifactStoreFactory.
type ArtifactStore interface {
	CreateFile(ctx context.Context, opts ArtifactOpts, localPath string) (*JobArtifact, error)
	CreateReader(ctx context.Context, opts ArtifactOpts, r io.Reader) (*JobArtifact, error)
}

// ArtifactReader is the read side used by the jobserver to list and look up
// artifacts. It is not scoped to a single job; callers pass the job id / row id.
type ArtifactReader interface {
	// ListByJob returns jobID's artifacts, newest first.
	ListByJob(ctx context.Context, jobID string) ([]*JobArtifact, error)
	// GetByID returns one row, or ErrArtifactNotFound.
	GetByID(ctx context.Context, artifactID int) (*JobArtifact, error)
}

// ArtifactStoreFactory produces a per-job ArtifactStore and also serves the read
// API. The job middleware calls For(...) once per execution to bind the artifact
// identity; the jobserver uses the read methods directly.
type ArtifactStoreFactory interface {
	ArtifactReader
	For(jobID, userID, kind string) ArtifactStore
}
