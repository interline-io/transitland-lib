package fetch

import (
	"fmt"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/rt/pb"
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
	if err != nil {
		log.Error().Err(err).Msg("fatal error during rt fetch")
	}
	ret.Result = result
	ret.Error = err
	return ret, err
}
