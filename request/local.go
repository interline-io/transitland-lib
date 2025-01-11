package request

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
)

func init() {
	var _ Downloader = &Local{}
	var _ DownloaderAll = &Local{}
	var _ Uploader = &Local{}
	var _ UploaderAll = &Local{}
}

type Local struct {
	Directory string
}

func (r Local) Download(ctx context.Context, ustr string, secret dmfr.Secret, auth dmfr.FeedAuthorization) (io.ReadCloser, int, error) {
	rd, err := os.Open(strings.TrimPrefix(filepath.Join(r.Directory, ustr), "file://"))
	return rd, 0, err
}

func (r Local) Upload(ctx context.Context, key string, secret dmfr.Secret, uploadFile io.Reader) error {
	// Do not overwrite files
	fn := filepath.Join(r.Directory, key)
	fmt.Println("Checking:", fn, "Dir:", r.Directory)
	out, err := os.OpenFile(fn, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		fmt.Println("failed to open:", err)
		return err
	}
	_, err = io.Copy(out, uploadFile)
	return err
}

func (r Local) DownloadFile(ctx context.Context, key string, fn string, secret dmfr.Secret) error {
	outf, err := os.Create(fn)
	if err != nil {
		return err
	}
	defer outf.Close()
	rio, _, err := r.Download(ctx, key, dmfr.Secret{}, dmfr.FeedAuthorization{})
	if err != nil {
		return err
	}
	if _, err := io.Copy(outf, rio); err != nil {
		return err
	}
	log.Info().Msgf("copied: '%s' -> '%s'", key, fn)
	return nil
}

func (r Local) UploadFile(ctx context.Context, fn string, key string, secret dmfr.Secret) error {
	inf, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer inf.Close()
	if err := r.Upload(ctx, key, dmfr.Secret{}, inf); err != nil {
		return err
	}
	log.Info().Msgf("copied: '%s' -> '%s'", fn, key)
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

func (r *Local) DownloadAll(ctx context.Context, outDir string, prefix string, secret dmfr.Secret, checkFile func(string) bool) ([]string, error) {
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
		if err := r.DownloadFile(ctx, relKey, outfn, secret); err != nil {
			return nil, err
		}
		// Ok
		ret = append(ret, outfn)
	}
	return ret, nil
}

func (r *Local) UploadAll(ctx context.Context, srcDir string, prefix string, secret dmfr.Secret, checkFile func(string) bool) error {
	// Get matching files
	fns, err := findFiles(srcDir, checkFile)
	if err != nil {
		return err
	}
	for _, fn := range fns {
		// Get relative location
		uploadKey := stripDir(srcDir, fn)
		uploadKey = filepath.Join(prefix, uploadKey)
		// Upload to relative location
		if err := r.UploadFile(ctx, fn, uploadKey, secret); err != nil {
			return err
		}
	}
	return nil
}
