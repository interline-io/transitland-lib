package jobserver

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
)

// fakeByteStore is a minimal request.Store (no Presigner) backed by a byte
// slice — exercises serveArtifact's streaming branch.
type fakeByteStore struct{ data []byte }

func (s *fakeByteStore) Download(ctx context.Context, key string) (io.ReadCloser, int, error) {
	return io.NopCloser(bytes.NewReader(s.data)), len(s.data), nil
}
func (s *fakeByteStore) DownloadAuth(ctx context.Context, key string, _ dmfr.FeedAuthorization) (io.ReadCloser, int, error) {
	return s.Download(ctx, key)
}
func (s *fakeByteStore) Upload(ctx context.Context, key string, r io.Reader) error { return nil }
func (s *fakeByteStore) ListKeys(ctx context.Context, prefix string) ([]string, error) {
	return nil, nil
}
func (s *fakeByteStore) SetSecret(dmfr.Secret) error { return nil }

// fakePresignStore additionally implements request.Presigner — exercises the
// 302 redirect branch.
type fakePresignStore struct {
	fakeByteStore
	signedURL      string
	gotKey         string
	gotDisposition string
}

func (s *fakePresignStore) CreateSignedUrl(ctx context.Context, key, contentDisposition string) (string, error) {
	s.gotKey = key
	s.gotDisposition = contentDisposition
	return s.signedURL, nil
}

var (
	_ request.Store     = (*fakeByteStore)(nil)
	_ request.Store     = (*fakePresignStore)(nil)
	_ request.Presigner = (*fakePresignStore)(nil)
)

func TestServeArtifactPresignRedirect(t *testing.T) {
	store := &fakePresignStore{signedURL: "https://signed.example/blob?sig=abc"}
	art := &model.JobArtifact{
		Filename:    "out.txt",
		ContentType: "text/plain",
		StorageKey:  "job-artifacts/job-1/uuid/out.txt",
		SizeBytes:   5,
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/download", nil)

	serveArtifact(rec, req, store, art)

	assert.Equal(t, http.StatusFound, rec.Code)
	assert.Equal(t, store.signedURL, rec.Header().Get("Location"))
	assert.Empty(t, rec.Body.String(), "presign redirect must not stream the body")
	// The verbatim storage key is presigned, and the sanitized filename rides
	// along as the content-disposition (not the raw body).
	assert.Equal(t, art.StorageKey, store.gotKey)
	assert.Contains(t, store.gotDisposition, "out.txt")
}

func TestServeArtifactStream(t *testing.T) {
	store := &fakeByteStore{data: []byte("hello")}
	art := &model.JobArtifact{
		Filename:    "out.txt",
		ContentType: "text/plain",
		StorageKey:  "job-artifacts/job-1/uuid/out.txt",
		SizeBytes:   5,
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/download", nil)

	serveArtifact(rec, req, store, art)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello", rec.Body.String())
	assert.Equal(t, "text/plain", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Header().Get("Content-Disposition"), "out.txt")
	assert.Equal(t, "5", rec.Header().Get("Content-Length"))
}
