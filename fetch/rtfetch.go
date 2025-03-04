package fetch

import (
	"context"
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

func RTFetch(ctx context.Context, atx tldb.Adapter, opts Options) (RTFetchResult, error) {
	cb := &rtFetchValidator{}
	result, err := fetchMain(ctx, atx, opts, cb)
	if err != nil {
		log.For(ctx).Error().Err(err).Msg("fatal error during rt fetch")
	}
	cb.ret.Result = result
	cb.ret.Error = err
	return cb.ret, err
}

type rtFetchValidator struct {
	ret RTFetchResult
}

func (r *rtFetchValidator) ValidateResponse(ctx context.Context, atx tldb.Adapter, fr request.FetchResponse, opts Options) (validationResponse, error) {
	// Validate
	v := validationResponse{}
	v.UploadTmpfile = fr.Filename
	v.UploadFilename = fmt.Sprintf("%s.pb", fr.ResponseSHA1)
	v.Found = false
	r.ret.Message, v.Error = rt.ReadFile(fr.Filename)
	return v, nil
}
