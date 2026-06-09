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

// serveArtifact serves art's bytes from an already-resolved store: a presigned
// redirect when the store supports it (S3/Azure), else a stream (Local). It takes
// art.StorageKey verbatim — unlike rest.serveFromStorage, which builds the key
// itself. The caller resolves the store, so this is testable with fakes.
func serveArtifact(w http.ResponseWriter, req *http.Request, store request.Store, art *model.JobArtifact) {
	ctx := req.Context()
	// art.Filename was sanitized at create time, so it is safe in a
	// Content-Disposition header / SAS disposition.
	if p, ok := store.(request.Presigner); ok {
		// Presigners build the disposition header themselves from a bare
		// filename (see request.Az), so pass the name, not a full header.
		signedURL, err := p.CreateSignedUrl(ctx, art.StorageKey, art.Filename)
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
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", art.Filename))
	if art.SizeBytes > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(art.SizeBytes, 10))
	}
	if _, err := io.Copy(w, rdr); err != nil {
		// Headers are already sent; can't change status. Log for diagnosis.
		log.For(ctx).Error().Err(err).Str("storage_key", art.StorageKey).Msg("artifact stream copy failed")
	}
}
