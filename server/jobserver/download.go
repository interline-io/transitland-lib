package jobserver

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/server/model"
)

// serveArtifact serves the artifact bytes from an already-resolved store,
// preferring a presigned redirect when the store supports it (S3/Azure) and
// falling back to streaming (Local). It takes art.StorageKey verbatim — unlike
// rest.serveFromStorage, which builds the key itself — and always sets download
// headers on the streaming path. It writes its own error responses and logs
// failures. The caller resolves the store (request.GetStore) so this is
// directly testable with a fake Presigner / byte store.
func serveArtifact(w http.ResponseWriter, req *http.Request, store request.Store, art *model.JobArtifact) {
	ctx := req.Context()
	// art.Filename was sanitized at create time, so it is safe to use in a
	// Content-Disposition header and as the presign content-disposition.
	disposition := fmt.Sprintf("attachment; filename=%q", art.Filename)
	if p, ok := store.(request.Presigner); ok {
		signedURL, err := p.CreateSignedUrl(ctx, art.StorageKey, disposition)
		if err != nil {
			internalError(w, req, "artifact presign failed", err)
			return
		}
		w.Header().Set("Location", signedURL)
		w.WriteHeader(http.StatusFound)
		return
	}
	rdr, _, err := store.Download(ctx, art.StorageKey)
	if err != nil {
		internalError(w, req, "artifact download failed", err)
		return
	}
	defer rdr.Close()
	w.Header().Set("Content-Type", art.ContentType)
	w.Header().Set("Content-Disposition", disposition)
	if art.SizeBytes > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(art.SizeBytes, 10))
	}
	if _, err := io.Copy(w, rdr); err != nil {
		// Headers are already sent; can't change status. Log for diagnosis.
		log.For(ctx).Error().Err(err).Str("storage_key", art.StorageKey).Msg("artifact stream copy failed")
	}
}
