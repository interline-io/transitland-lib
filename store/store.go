package store

import (
	"context"
	"errors"
	"io"
	"net/url"
	"os"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/request"
	"github.com/interline-io/transitland-lib/tlcsv"
)

// Store provides data storage methods.
type Store interface {
	request.Uploader
	request.Downloader
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
		s, storeErr = request.NewS3FromUrl(ustr)
	case "az":
		s, storeErr = request.NewAzFromUrl(ustr)
	case "file":
		s = request.Local{Directory: ustr}
	default:
		if ustr == "" {
			return nil, errors.New("no storage specified")
		} else {
			s = request.Local{Directory: ustr}
		}
	}
	return s, storeErr
}

// Download is a convenience method for downloading a file from the store.
func Download(storage string, key string) (io.ReadCloser, error) {
	log.Debug().Str("src", key).Str("storage", storage).Msg("fetch: download from store")
	st, err := GetStore(storage)
	if err != nil {
		return nil, err
	}
	r, _, err := st.Download(context.Background(), key, tl.Secret{}, tl.FeedAuthorization{})
	if err != nil {
		return nil, err
	}
	return r, nil
}

// UploadFile is a convenience method for uploading a file to the store.
func UploadFile(storage string, src string, dst string) error {
	log.Debug().Str("src", src).Str("storage", storage).Str("storage_key", dst).Msg("fetch: upload to store")
	rp, err := os.Open(src)
	if err != nil {
		return err
	}
	defer rp.Close()
	st, err := GetStore(storage)
	if err != nil {
		return err
	}
	if err := st.Upload(context.Background(), dst, tl.Secret{}, rp); err != nil {
		return err
	}
	return nil
}

// NewStoreAdapter is a convenience method for getting a GTFS Zip reader from the store.
func NewStoreAdapter(storage string, key string, fragment string) (*tlcsv.TmpZipAdapter, error) {
	r, err := Download(storage, key)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return tlcsv.NewTmpZipAdapterFromReader(r, fragment)
}
