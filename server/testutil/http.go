package testutil

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"github.com/interline-io/log"
)

func NewTestServer(baseDir string) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Trace().Str("baseDir", baseDir).Msgf("request: %s %s", r.Method, r.URL.Path)
		rootDir, err := os.OpenRoot(baseDir)
		if err != nil {
			http.Error(w, "internal server error", 500)
			return
		}
		defer rootDir.Close()
		// Remove leading slash from the URL path
		// so it can be used to open files in the base directory.
		p := strings.TrimPrefix(r.URL.Path, "/")
		f, err := rootDir.Open(p)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}
		buf, err := io.ReadAll(f)
		if err != nil {
			http.Error(w, "internal server error", 500)
			return
		}
		w.Write(buf)
	}))
	return ts
}
