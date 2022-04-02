package fetch

import (
	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
)

func RTFetch(atx tldb.Adapter, feed tl.Feed, opts Options) (Result, error) {
	cb := func(filename string) (validationResponse, error) {
		// Validate
		v := validationResponse{}
		v.Filename = filename
		v.Found = false
		_, v.Error = rt.ReadFile(filename)
		return v, nil
	}
	return ffetch(atx, feed, opts, cb)
}
