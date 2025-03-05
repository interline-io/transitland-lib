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
	cb := &RTFetchValidator{}
	result, err := Fetch(ctx, atx, opts, cb)
	if err != nil {
		log.For(ctx).Error().Err(err).Msg("fatal error during rt fetch")
	}
	cb.Result.Result = result
	cb.Result.Error = err
	return cb.Result, err
}

type RTFetchValidator struct {
	Result RTFetchResult
}

func (r *RTFetchValidator) ValidateResponse(ctx context.Context, atx tldb.Adapter, fn string, fr request.FetchResponse, opts Options) (FetchValidationResult, error) {
	// Validate
	v := FetchValidationResult{}
	v.UploadTmpfile = fn
	v.UploadFilename = fmt.Sprintf("%s.pb", fr.ResponseSHA1)
	v.Found = false
	r.Result.Message, v.Error = rt.ReadFile(fn)
	return v, nil
}
