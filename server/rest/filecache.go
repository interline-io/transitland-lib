package rest

import (
	"io"
	"os"

	"github.com/hypirion/go-filecache"
	"github.com/interline-io/log"
)

// local file cache
var localFileCache *filecache.Filecache

// Set config
func init() {
	cachesize := 100
	d := &noCache{}
	if dc, err := filecache.New(filecache.Size(cachesize)*filecache.MiB, d); err == nil {
		localFileCache = dc
	} else {
		log.Fatal().Msgf("Error creating local file cache: %s", err.Error())
	}
}

// noCache is a dummy cache
type noCache struct {
}

func (d *noCache) Has(key string) (bool, error) {
	return false, nil
}

func (d *noCache) Get(dst io.Writer, key string) error {
	return os.ErrNotExist
}

func (d *noCache) Put(key string, src io.Reader) error {
	return nil
}
