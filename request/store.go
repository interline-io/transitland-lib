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

// Store provides data storage methods.
type Store interface {
	Uploader
	Downloader
}

// GetStore returns a configured store based on the provided url.
func GetStore(ustr string) (Store, error) {
	u, err := url.Parse(ustr)
	if err != nil {
		return nil, err
	}
	var storeErr error
	var s Store
	switch u.Scheme {
	case "s3":
		s, storeErr = NewS3FromUrl(ustr)
	case "az":
		s, storeErr = NewAzFromUrl(ustr)
	case "file":
		s = Local{Directory: ustr}
	default:
		if ustr == "" {
			return nil, errors.New("no storage specified")
		} else {
			s = Local{Directory: ustr}
		}
	}
	return s, storeErr
}

// Download is a convenience method for downloading a file from the store.
func Download(ctx context.Context, storage string, key string) (io.ReadCloser, error) {
	log.For(ctx).Debug().Str("src", key).Str("storage", storage).Msg("fetch: download from store")
	st, err := GetStore(storage)
	if err != nil {
		return nil, err
	}
	r, _, err := st.Download(ctx, key, dmfr.Secret{}, dmfr.FeedAuthorization{})
	if err != nil {
		return nil, err
	}
	return r, nil
}

// UploadFile is a convenience method for uploading a file to the store.
func UploadFile(ctx context.Context, storage string, src string, dst string) error {
	log.For(ctx).Debug().Str("src", src).Str("storage", storage).Str("storage_key", dst).Msg("fetch: upload to store")
	rp, err := os.Open(src)
	if err != nil {
		return err
	}
	defer rp.Close()
	st, err := GetStore(storage)
	if err != nil {
		return err
	}
	if err := st.Upload(ctx, dst, dmfr.Secret{}, rp); err != nil {
		return err
	}
	return nil
}
