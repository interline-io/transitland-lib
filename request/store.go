package request

import (
	"context"
	"errors"
	"io"
	"net/url"
	"os"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
)

type Store interface {
	Uploader
	Downloader
}

type CanSetSecret interface {
	SetSecret(dmfr.Secret) error
}

type Bucket interface {
	CanSetSecret
	Downloader
	Uploader
	DownloadAll(ctx context.Context, outDir string, prefix string, checkFile func(string) bool) ([]string, error)
	UploadAll(ctx context.Context, srcDir string, prefix string, checkFile func(string) bool) error
}

type Presigner interface {
	CreateSignedUrl(context.Context, string, string) (string, error)
}

// GetStore returns a configured store based on the provided url.
func GetStore(ustr string) (Store, error) {
	var h Store
	h, err := GetBucket(ustr)
	if err != nil {
		return nil, err
	}
	return h, nil
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

func copyToFile(ctx context.Context, rio io.Reader, outfn string) error {
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

func DownloadFileHelper(r Downloader, ctx context.Context, key string, fn string) error {
	rio, _, err := r.Download(ctx, key)
	if err != nil {
		return err
	}
	defer rio.Close()
	return copyToFile(ctx, rio, fn)
}

func UploadFileHelper(r Uploader, ctx context.Context, fn string, key string) error {
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
