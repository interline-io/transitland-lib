package request

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
)

var _ Store = (*Local)(nil)

type Local struct {
	Directory string
}

func (r *Local) SetSecret(secret dmfr.Secret) error {
	return nil
}

func (r Local) Exists(ctx context.Context, key string) bool {
	fn := filepath.Join(r.Directory, key)
	info, err := os.Stat(fn)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func (r Local) Download(ctx context.Context, key string) (io.ReadCloser, int, error) {
	key = strings.TrimPrefix(key, "file://")
	fn := filepath.Join(r.Directory, key)
	log.Debug().Msgf("local store: downloading key '%s', full path is '%s'", key, fn)
	rd, err := os.Open(fn)
	if err != nil {
		return nil, 0, err
	}
	return rd, 0, nil
}

func (r Local) DownloadAuth(ctx context.Context, key string, auth dmfr.FeedAuthorization) (io.ReadCloser, int, error) {
	return r.Download(ctx, key)
}

func (r *Local) ListKeys(ctx context.Context, prefix string) ([]string, error) {
	// Get matching files
	downloadKeys, err := findFiles(r.Directory, func(b string) bool {
		return strings.HasPrefix(b, prefix)
	})
	if err != nil {
		return nil, err
	}
	var ret []string
	for _, key := range downloadKeys {
		ret = append(ret, stripDir(r.Directory, key))
	}
	return ret, nil
}

func (r Local) Upload(ctx context.Context, key string, uploadFile io.Reader) error {
	outfn := filepath.Join(r.Directory, key)
	log.Debug().Msgf("local store: uploading to key '%s', full path is '%s'", key, outfn)
	// Check if directory exists
	if _, err := mkdir(filepath.Dir(outfn), ""); err != nil {
		return err
	}
	// Write to a temp file in the same directory, then atomically rename into
	// place: an interrupted write never leaves a corrupt object at the key, and
	// the rename overwrites idempotently — required for content-addressed retries,
	// where a previous failed upload must not block re-uploading the same key.
	tmpf, err := os.CreateTemp(filepath.Dir(outfn), ".upload-*")
	if err != nil {
		return err
	}
	tmpName := tmpf.Name()
	defer os.Remove(tmpName) // no-op once renamed away; cleans up on any failure
	if _, err := io.Copy(tmpf, uploadFile); err != nil {
		tmpf.Close()
		return err
	}
	if err := tmpf.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, outfn)
}
