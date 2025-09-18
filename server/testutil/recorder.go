package testutil

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/interline-io/log"
	"gopkg.in/dnaeon/go-vcr.v2/cassette"
	"gopkg.in/dnaeon/go-vcr.v2/recorder"
)

// NewRecorder returns a configured recorder.
// It doesn't support absolute paths, so can't use testutil.RelPath()
// Must be relative to test directory, not project root.
func NewRecorder(path string, replaceUrl string) *recorder.Recorder {
	// Start our recorder
	r, err := recorder.New(path)
	if err != nil {
		log.Fatal().Err(err).Msgf("could not open recorder path '%s'", path)
	}
	r.SetMatcher(func(r *http.Request, i cassette.Request) bool {
		if r.Body == nil {
			return cassette.DefaultMatcher(r, i)
		}
		var b bytes.Buffer
		if _, err := b.ReadFrom(r.Body); err != nil {
			return false
		}
		r.Body = ioutil.NopCloser(&b)
		// Check default
		if ok := cassette.DefaultMatcher(r, i) && (b.String() == "" || b.String() == i.Body); ok {
			return true
		}
		// Check on hashed url
		if replaceUrl != "" {
			r.URL, _ = url.Parse(replaceUrl)
		}
		return cassette.DefaultMatcher(r, i)
	})
	r.AddFilter(func(i *cassette.Interaction) error {
		// Hash url and zap headers
		if replaceUrl != "" {
			i.URL = replaceUrl
		}
		i.Request.Headers = nil
		i.Response.Headers = nil
		return nil
	})
	return r
}
