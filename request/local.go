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

func init() {
	var _ Bucket = &Local{}
}

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

func (r Local) Download(ctx context.Context, ustr string) (io.ReadCloser, int, error) {
	fn := strings.TrimPrefix(filepath.Join(r.Directory, ustr), "file://")
	log.Debug().Msgf("local store: downloading key '%s', full path is '%s'", ustr, fn)
	rd, err := os.Open(fn)
	if err != nil {
		return nil, 0, err
	}
	return rd, 0, nil
}

func (r Local) DownloadAuth(ctx context.Context, ustr string, auth dmfr.FeedAuthorization) (io.ReadCloser, int, error) {
	return r.Download(ctx, ustr)
}

func (r *Local) DownloadAll(ctx context.Context, outDir string, prefix string, checkFile func(string) bool) ([]string, error) {
	// Get matching files
	downloadKeys, err := findFiles(r.Directory, func(b string) bool {
		if !strings.HasPrefix(b, prefix) {
			return false
		}
		if checkFile != nil {
			return checkFile(b)
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	var ret []string
	for _, key := range downloadKeys {
		// Get relative location
		relKey := stripDir(r.Directory, key)
		// Get output location and create directory
		outfn := filepath.Join(outDir, stripDir(prefix, relKey))
		if _, err := mkdir(filepath.Dir(outfn), ""); err != nil {
			return nil, err
		}
		// Download to output file
		if err := DownloadFileHelper(r, ctx, relKey, outfn); err != nil {
			return nil, err
		}
		// Ok
		ret = append(ret, outfn)
	}
	return ret, nil
}

func (r Local) Upload(ctx context.Context, key string, uploadFile io.Reader) error {
	outfn := filepath.Join(r.Directory, key)
	log.Debug().Msgf("s3 store: uploading to key '%s', full path is '%s'", key, outfn)
	// Check if directory exists
	if _, err := mkdir(filepath.Dir(outfn), ""); err != nil {
		return err
	}
	// Do not overwrite files
	outf, err := os.OpenFile(outfn, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}
	n, err := io.Copy(outf, uploadFile)
	log.Debug().Msgf("local store: wrote %d bytes to '%s'", n, outfn)
	return err
}

func (r *Local) UploadAll(ctx context.Context, srcDir string, prefix string, checkFile func(string) bool) error {
	// Get matching files
	fns, err := findFiles(srcDir, checkFile)
	if err != nil {
		return err
	}
	for _, fn := range fns {
		// Get relative location
		uploadKey := filepath.Join(prefix, stripDir(srcDir, fn))
		// Upload to relative location
		if err := UploadFileHelper(r, ctx, fn, uploadKey); err != nil {
			return err
		}
	}
	return nil
}
