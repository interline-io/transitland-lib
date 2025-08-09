package gbfs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/interline-io/transitland-lib/tt"
)

// Serve a directory of GBFS files. Used for testing.
type TestGbfsServer struct {
	Language string
	Path     string
}

func (g *TestGbfsServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data, err := g.open(r.Host, r.URL.Path)
	if err != nil {
		w.WriteHeader(404)
	}
	w.Write(data)
}

func (g *TestGbfsServer) open(host string, path string) ([]byte, error) {
	if path == "/" || path == "" || path == "/gbfs.json" {
		sf := SystemFile{}
		fis, err := os.ReadDir(g.Path)
		_ = err
		var sfs SystemFeeds
		for _, fi := range fis {
			if strings.HasSuffix(fi.Name(), ".json") {
				fn := strings.Replace(fi.Name(), ".json", "", -1)
				url := fmt.Sprintf("http://%s/%s.json", host, fn)
				sfs.Feeds = append(sfs.Feeds, &SystemFeed{Name: tt.NewString(fn), URL: tt.NewString(url)})
			}
		}
		sf.Data = map[string]*SystemFeeds{}
		sf.Data[g.Language] = &sfs
		data, err := json.Marshal(sf)
		return data, err
	}
	r, err := os.Open(filepath.Join(g.Path, path))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(r)
}
