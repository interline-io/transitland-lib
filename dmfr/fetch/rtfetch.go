package fetch

import (
	"fmt"

	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/request"
	"github.com/interline-io/transitland-lib/tldb"
)

func RTFetch(atx tldb.Adapter, feed tl.Feed, opts Options) (Result, error) {
	cb := func(fr request.FetchResponse) (validationResponse, error) {
		// Validate
		v := validationResponse{}
		v.Filename = fr.Filename
		v.UploadFilename = fmt.Sprintf("%s.pb", fr.ResponseSHA1)
		v.Found = false
		_, v.Error = rt.ReadFile(fr.Filename)
		return v, nil
	}
	return ffetch(atx, feed, opts, cb)
}
