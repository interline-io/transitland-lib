package request

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

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
	rd, err := os.Open(strings.TrimPrefix(filepath.Join(r.Directory, ustr), "file://"))
	return rd, 0, err
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
		if err := DownloadFile(r, ctx, relKey, outfn); err != nil {
			return nil, err
		}
		// Ok
		ret = append(ret, outfn)
	}
	return ret, nil
}
func (r Local) Upload(ctx context.Context, key string, uploadFile io.Reader) error {
	// Check if directory exists
	fn := filepath.Join(r.Directory, key)
	if _, err := mkdir(filepath.Dir(fn), ""); err != nil {
		return err
	}
	// Do not overwrite files
	out, err := os.OpenFile(fn, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, uploadFile)
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
		if err := UploadFile(r, ctx, fn, uploadKey); err != nil {
			return err
		}
	}
	return nil
}
