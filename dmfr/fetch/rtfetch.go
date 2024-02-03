package fetch

import (
	"fmt"

	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/tl/request"
	"github.com/interline-io/transitland-lib/tldb"
)

type RTFetchResult struct {
	Message *pb.FeedMessage
	Result
}

func RTFetch(atx tldb.Adapter, opts Options) (RTFetchResult, error) {
	ret := RTFetchResult{}
	cb := func(fr request.FetchResponse) (validationResponse, error) {
		// Validate
		v := validationResponse{}
		v.UploadTmpfile = fr.Filename
		v.UploadFilename = fmt.Sprintf("%s.pb", fr.ResponseSHA1)
		v.Found = false
		ret.Message, v.Error = rt.ReadFile(fr.Filename)
		return v, nil
	}
	result, err := ffetch(atx, opts, cb)
	ret.Result = result
	return ret, err
}
