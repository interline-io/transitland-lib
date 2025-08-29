package gbfs

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"

	"github.com/interline-io/transitland-lib/tt"
)

// Serve a directory of GBFS files. Used for testing.
type TestGbfsServer struct {
	Language string
	Path     string
	fsys     fs.FS
}

// NewTestGbfsServer creates a new TestGbfsServer with a rooted filesystem
func NewTestGbfsServer(language, path string) *TestGbfsServer {
	return &TestGbfsServer{
		Language: language,
		Path:     path,
		fsys:     os.DirFS(path),
	}
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
		fis, err := fs.ReadDir(g.fsys, ".")
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
	// Clean the path and ensure it doesn't escape the root
	cleanPath := strings.TrimPrefix(path, "/")
	r, err := g.fsys.Open(cleanPath)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
