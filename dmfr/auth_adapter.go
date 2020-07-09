package dmfr

import (
	"errors"
	"net/url"
	"os"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/gtcsv"
	"github.com/interline-io/gotransit/internal/log"
)

// AuthenticatedURLAdapter is similar to URLAdapter but takes auth and secrets.
type AuthenticatedURLAdapter struct {
	downloadtmp string
	gtcsv.ZipAdapter
}

// Download the URL to a temporary file and set the correct adapter
func (adapter *AuthenticatedURLAdapter) Download(address string, auth gotransit.FeedAuthorization, secret Secret) error {
	// Handle fragments
	u, err := url.Parse(address)
	if err != nil {
		return err
	}
	// Download feed
	tmpfile, err := AuthenticatedRequest(address, secret, auth)
	if err != nil {
		return err
	}
	adapter.downloadtmp = tmpfile
	if u.Fragment != "" {
		tmpfile = tmpfile + "#" + u.Fragment
	}
	za := gtcsv.NewZipAdapter(tmpfile)
	if za == nil {
		return errors.New("could not open")
	}
	adapter.ZipAdapter = *za
	return nil
}

// Close the adapter, and remove the temporary file. An error is returned if the file could not be deleted.
func (adapter *AuthenticatedURLAdapter) Close() error {
	if adapter.downloadtmp != "" {
		log.Debug("Removing temp file: %s", adapter.downloadtmp)
		if err := os.Remove(adapter.downloadtmp); err != nil {
			return err
		}
	}
	return adapter.ZipAdapter.Close()
}
