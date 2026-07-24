// Package artifactstore persists job artifacts (a blob in request.Store plus a
// tl_job_artifacts row). It follows validator.SaveValidationReport: upload the
// blob first and insert the row second, so a failure can't leave a row pointing
// at nothing.
package artifactstore

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tldb/postgres"
	sq "github.com/irees/squirrel"
)

// artifactPrefix namespaces all artifact keys within the storage backend so
// they never collide with feed-version ({sha1}.zip) or validation-report keys.
const artifactPrefix = "job-artifacts"

const defaultContentType = "application/octet-stream"

// Store is the unscoped factory and read side.
type Store struct {
	dbx        tldb.Ext
	storageURL string
}

var (
	_ model.ArtifactStoreFactory = (*Store)(nil)
	_ model.ArtifactDeleter      = (*Store)(nil)
	_ model.ArtifactStore        = (*scoped)(nil)
)

// NewStore binds the store to a db handle and the resolved Config.ArtifactStorage
// URL. An empty URL is allowed; writes then fail loudly (see create) rather than
// falling back to feed-version storage.
func NewStore(dbx tldb.Ext, storageURL string) *Store {
	return &Store{dbx: dbx, storageURL: storageURL}
}

// For binds the store to one job's identity so a worker can publish files
// attributed to it.
func (s *Store) For(jobID, userID, kind string) model.ArtifactStore {
	return &scoped{store: s, jobID: jobID, userID: userID, kind: kind}
}

// ListByJob returns the live (not soft-deleted) artifacts produced by jobID,
// newest first.
func (s *Store) ListByJob(ctx context.Context, jobID string) ([]*model.JobArtifact, error) {
	q := sq.Select("*").From("tl_job_artifacts").Where(sq.Eq{"job_id": jobID, "deleted_at": nil}).OrderBy("id desc")
	var ret []*model.JobArtifact
	if err := dbutil.Select(ctx, s.dbx, q, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}

// GetByID returns a single live (not soft-deleted) artifact row, or
// ErrArtifactNotFound.
func (s *Store) GetByID(ctx context.Context, artifactID int) (*model.JobArtifact, error) {
	q := sq.Select("*").From("tl_job_artifacts").Where(sq.Eq{"id": artifactID, "deleted_at": nil}).Limit(1)
	var ret []*model.JobArtifact
	if err := dbutil.Select(ctx, s.dbx, q, &ret); err != nil {
		return nil, err
	}
	if len(ret) == 0 {
		return nil, model.ErrArtifactNotFound
	}
	return ret[0], nil
}

// SoftDelete flags an artifact row deleted (deleted_at = now()); its stored bytes
// are left for a later blob-culling pass, and reads skip soft-deleted rows.
// Deleting an already-deleted or missing row is a no-op.
func (s *Store) SoftDelete(ctx context.Context, artifactID int) error {
	q := sq.Update("tl_job_artifacts").
		Set("deleted_at", time.Now().UTC()).
		Where(sq.Eq{"id": artifactID, "deleted_at": nil})
	return dbutil.Update(ctx, s.dbx, q)
}

// scoped is an ArtifactStore bound to one job's identity.
type scoped struct {
	store  *Store
	jobID  string
	userID string
	kind   string
}

// CreateFile uploads localPath and records an artifact row for the current job.
func (s *scoped) CreateFile(ctx context.Context, opts model.ArtifactOpts, localPath string) (*model.JobArtifact, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("artifactstore: open %q: %w", localPath, err)
	}
	defer f.Close()
	return s.create(ctx, opts, f)
}

// CreateReader uploads from r and records an artifact row for the current job.
func (s *scoped) CreateReader(ctx context.Context, opts model.ArtifactOpts, r io.Reader) (*model.JobArtifact, error) {
	return s.create(ctx, opts, r)
}

func (s *scoped) create(ctx context.Context, opts model.ArtifactOpts, r io.Reader) (*model.JobArtifact, error) {
	// A job id is required to address the artifact; backends that never assign
	// one (fire-and-forget Redis) fail loudly here rather than write an orphan.
	if s.jobID == "" {
		return nil, errors.New("artifactstore: cannot create artifact without a job id (backend does not track jobs)")
	}
	filename := sanitizeFilename(opts.Filename)
	if filename == "" {
		return nil, fmt.Errorf("artifactstore: invalid filename %q", opts.Filename)
	}
	// No fallback to feed-version storage: fail rather than write somewhere unexpected.
	if s.store.storageURL == "" {
		return nil, errors.New("artifactstore: no artifact storage configured")
	}
	store, err := request.GetStore(s.store.storageURL)
	if err != nil {
		return nil, fmt.Errorf("artifactstore: get storage: %w", err)
	}
	// Key is namespaced by job id and carries a uuid segment so re-emitting the
	// same filename never collides (the Local store opens with O_EXCL) and keys
	// are not guessable from the job id alone.
	key := path.Join(artifactPrefix, s.jobID, uuid.NewString(), filename)

	// Capture size + sha1 in one pass while streaming to storage.
	h := sha1.New()
	counter := &countingWriter{}
	tee := io.TeeReader(r, io.MultiWriter(h, counter))
	if err := store.Upload(ctx, key, tee); err != nil {
		return nil, fmt.Errorf("artifactstore: upload %q: %w", key, err)
	}

	art := &model.JobArtifact{
		JobID:       s.jobID,
		JobKind:     s.kind,
		UserID:      s.userID,
		Filename:    filename,
		ContentType: contentTypeOrDefault(opts.ContentType),
		SizeBytes:   counter.n,
		SHA1:        hex.EncodeToString(h.Sum(nil)),
		StorageKey:  key,
	}
	// Upload-then-insert (mirrors SaveValidationReport): upload is the last
	// failable step before the row, so a failed insert leaves at most an orphan
	// blob, never a row that points at nothing.
	atx := postgres.NewPostgresAdapterFromDBX(s.store.dbx)
	if _, err := atx.Insert(ctx, art); err != nil {
		log.For(ctx).Error().Err(err).Str("job_id", s.jobID).Str("storage_key", key).Msg("failed to insert job artifact row")
		return nil, fmt.Errorf("artifactstore: insert row: %w", err)
	}
	log.For(ctx).Info().Str("job_id", s.jobID).Int("artifact_id", art.ID).Str("filename", art.Filename).Int64("size_bytes", art.SizeBytes).Msg("job artifact saved")
	return art, nil
}

// countingWriter tallies bytes written; paired with a hash via io.MultiWriter.
type countingWriter struct{ n int64 }

func (c *countingWriter) Write(p []byte) (int, error) {
	c.n += int64(len(p))
	return len(p), nil
}

func contentTypeOrDefault(ct string) string {
	if ct = strings.TrimSpace(ct); ct != "" {
		return ct
	}
	return defaultContentType
}

// sanitizeFilename reduces a worker-supplied name to a safe basename. This is
// security-critical: the Local store joins the key onto a directory (a
// write-anywhere primitive otherwise) and the name is reused as the presigned
// Content-Disposition (header-injection otherwise). Returns "" if nothing safe
// remains, which callers reject.
func sanitizeFilename(name string) string {
	name = strings.TrimSpace(name)
	// Basename only: drop any directory components (handles / and \).
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	if name == "" || name == "." || name == ".." {
		return ""
	}
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9',
			r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	// No leading dots (hidden files / traversal remnants).
	out := strings.TrimLeft(b.String(), ".")
	return out
}
