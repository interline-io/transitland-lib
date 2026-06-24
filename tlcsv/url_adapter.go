package tlcsv

import (
	"context"
	"strings"

	"github.com/interline-io/transitland-lib/request"
)

// URLAdapter downloads a GTFS URL to a temporary file, and removes the file when it is closed.
type URLAdapter struct {
	url     string
	reqOpts []request.RequestOption
	ZipAdapter
}

func NewURLAdapter(address string, opts ...request.RequestOption) *URLAdapter {
	return &URLAdapter{
		url:     address,
		reqOpts: opts,
	}
}

func (adapter *URLAdapter) String() string {
	return adapter.url
}

// Open the adapter, and download the provided URL to a temporary file.
func (adapter *URLAdapter) Open() error {
	if adapter.ZipAdapter.path != "" {
		return nil // already open
	}
	// Remove and keep internal path prefix
	url, fragment, _ := strings.Cut(adapter.url, "#")
	// Download to temporary file
	tmpfile, fr, err := request.AuthenticatedRequestDownload(context.TODO(), url, adapter.reqOpts...)
	if err != nil {
		return err
	}
	if fr.FetchError != nil {
		return fr.FetchError
	}
	// Add internal path prefix back
	adapter.ZipAdapter = ZipAdapter{
		path:           tmpfile,
		internalPrefix: fragment,
		tmpfiles:       []string{tmpfile},
	}
	return adapter.ZipAdapter.Open()
}
