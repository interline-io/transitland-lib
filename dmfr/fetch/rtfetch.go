package fetch

import (
	"fmt"

	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/request"
	"github.com/interline-io/transitland-lib/tldb"
)

type RTFetchResult struct {
	Message *pb.FeedMessage
	Result
}

func RTFetch(atx tldb.Adapter, feed tl.Feed, opts Options) (*pb.FeedMessage, Result, error) {
	var msg *pb.FeedMessage
	cb := func(fr request.FetchResponse) (validationResponse, error) {
		// Validate
		v := validationResponse{}
		v.UploadTmpfile = fr.Filename
		v.UploadFilename = fmt.Sprintf("%s.pb", fr.ResponseSHA1)
		v.Found = false
		msg, v.Error = rt.ReadFile(fr.Filename)
		return v, nil
	}
	result, err := ffetch(atx, feed, opts, cb)
	return msg, result, err
}
