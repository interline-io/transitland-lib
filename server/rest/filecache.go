package rest

import (
	"fmt"
	"io"
	"os"

	"github.com/hypirion/go-filecache"
)

// local file cache
var localFileCache *filecache.Filecache

// Set config
func init() {
	cachesize := 100
	d := dirCache{}
	if dc, err := filecache.New(filecache.Size(cachesize)*filecache.MiB, &d); err == nil {
		localFileCache = dc
	} else {
		fmt.Println("Error creating local file cache:", err)
	}
}

// dirCache is a simple transient cache
type dirCache struct {
	path string
}

func (d *dirCache) Has(key string) (bool, error) {
	return false, nil
}

func (d *dirCache) Get(dst io.Writer, key string) error {
	return os.ErrNotExist
}

func (d *dirCache) Put(key string, src io.Reader) error {
	return nil
}
