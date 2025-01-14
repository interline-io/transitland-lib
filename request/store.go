package request

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
)

type CanSetSecret interface {
	SetSecret(dmfr.Secret) error
}

type Bucket interface {
	Downloader
	Uploader
	CanSetSecret
	ListAll(ctx context.Context, prefix string) ([]string, error)
}

type Presigner interface {
	CreateSignedUrl(context.Context, string, string) (string, error)
}

// GetBucket returns a configured bucket based on the provided url.
func GetBucket(ustr string) (Bucket, error) {
	u, err := url.Parse(ustr)
	if err != nil {
		return nil, err
	}
	var storeErr error
	var s Bucket
	switch u.Scheme {
	case "s3":
		s, storeErr = NewS3FromUrl(ustr)
	case "az":
		s, storeErr = NewAzFromUrl(ustr)
	case "file":
		s = &Local{Directory: ustr}
	default:
		if ustr == "" {
			return nil, errors.New("no storage specified")
		} else {
			s = &Local{Directory: ustr}
		}
	}
	return s, storeErr
}

func Download(ctx context.Context, r Downloader, key string, fn string) error {
	log.Debug().Msgf("store: downloading key '%s' to file '%s'", key, fn)
	rio, _, err := r.Download(ctx, key)
	if err != nil {
		return err
	}
	defer rio.Close()
	return copyToFile(ctx, rio, fn)
}

func Upload(ctx context.Context, r Uploader, fn string, key string) error {
	log.Debug().Msgf("store: uploading file '%s' to key '%s'", fn, key)
	inf, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer inf.Close()
	if err := r.Upload(ctx, key, inf); err != nil {
		return err
	}
	return nil
}

func UploadAll(ctx context.Context, r Uploader, srcDir string, prefix string, checkFile func(string) bool) error {
	fns, err := findFiles(srcDir, checkFile)
	if err != nil {
		return err
	}
	for _, fn := range fns {
		key := filepath.Join(prefix, stripDir(srcDir, fn))
		if err := Upload(ctx, r, fn, key); err != nil {
			return err
		}
	}
	return nil
}

func DownloadAll(ctx context.Context, r Bucket, outDir string, prefix string, checkFile func(string) bool) ([]string, error) {
	keys, err := r.ListAll(ctx, prefix)
	if err != nil {
		return nil, nil
	}
	var ret []string
	for _, key := range keys {
		outfn := filepath.Join(outDir, stripDir(prefix, key))
		if _, err := mkdir(filepath.Dir(outfn), ""); err != nil {
			return nil, err
		}
		if err := Download(ctx, r, key, outfn); err != nil {
			return nil, err
		}
		ret = append(ret, outfn)
	}
	return ret, nil
}

func copyToFile(_ context.Context, rio io.Reader, outfn string) error {
	log.Trace().Msgf("copyToFile: %s", outfn)
	outf, err := os.Create(outfn)
	if err != nil {
		return err
	}
	defer outf.Close()
	if _, err := io.Copy(outf, rio); err != nil {
		return nil
	}
	return nil
}

func mkdir(basePath string, path string) (string, error) {
	ret := basePath
	if path != "" {
		ret = filepath.Join(basePath, path)
	}
	if fi, err := os.Stat(ret); err == nil && fi.IsDir() {
		return ret, nil
	}
	log.Info().Msgf("mkdir '%s'", ret)
	if err := os.MkdirAll(ret, os.ModePerm|os.ModeDir); err != nil {
		return "", err
	}
	return ret, nil
}

func findFiles(srcDir string, checkFile func(string) bool) ([]string, error) {
	if checkFile == nil {
		checkFile = func(string) bool { return true }
	}
	var ret []string
	if err := filepath.Walk(srcDir, func(path string, info fs.FileInfo, err error) error {
		if info == nil {
			return errors.New("no file")
		}
		fn := info.Name()
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if info.Size() == 0 {
			return nil
		}
		if err != nil {
			return err
		}
		relFn := filepath.Join(stripDir(srcDir, path), fn)
		if checkFile != nil && !checkFile(relFn) {
			return nil
		}
		ret = append(ret, path)
		return nil
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

func stripDir(srcDir string, fn string) string {
	if !strings.HasSuffix(srcDir, "/") {
		srcDir = srcDir + "/"
	}
	return strings.TrimPrefix(fn, srcDir)
}
