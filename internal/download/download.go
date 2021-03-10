package download

import (
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"
)

var a io.ReadCloser

// TemporaryDownload saves a file to a temporary location and deletes it on Close().
type TemporaryDownload struct {
	URL string
	*os.File
}

// Open downloads to a temporary location.
func (td *TemporaryDownload) Open() error {
	u, err := url.Parse(td.URL)
	if err != nil {
		return err
	}
	if !(u.Scheme == "http" || u.Scheme == "https") {
		return errors.New("invalid url")
	}
	tmpfile, err := ioutil.TempFile("", "tlib")
	if err != nil {
		return err
	}
	td.File = tmpfile
	// Download HTTP
	client := &http.Client{
		Timeout: 600 * time.Second,
	}
	req, err := http.NewRequest("GET", td.URL, nil)
	if err != nil {
		return err
	}
	// TODO: DMFR-style auth
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if _, err := io.Copy(td.File, resp.Body); err != nil {
		return err
	}
	// Reset position
	td.File.Sync()
	td.File.Seek(0, 0)
	return nil
}

// Close closes and removes the underlying file.
func (td *TemporaryDownload) Close() error {
	if td.File == nil {
		return nil
	}
	td.File.Close()
	os.Remove(td.File.Name())
	td.File = nil
	return nil
}
