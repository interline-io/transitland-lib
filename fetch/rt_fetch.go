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

func RTFetch(ctx context.Context, atx tldb.Adapter, opts RTFetchOptions) (RTFetchResult, error) {
	r := NewRTFetchValidator(opts)
	return r.Fetch(ctx, atx)
}

type RTFetchResult struct {
	Message *pb.FeedMessage
	Result
}

type RTFetchOptions struct {
	Options
}

type RTFetchValidator struct {
	Result         RTFetchResult
	RTFetchOptions RTFetchOptions
}

func NewRTFetchValidator(opts RTFetchOptions) *RTFetchValidator {
	return &RTFetchValidator{RTFetchOptions: opts}
}

func (r *RTFetchValidator) Fetch(ctx context.Context, atx tldb.Adapter) (RTFetchResult, error) {
	result, err := Fetch(ctx, atx, r.RTFetchOptions.Options, r)
	if err != nil {
		log.For(ctx).Error().Err(err).Msg("fatal error during rt fetch")
	}
	r.Result.Result = result
	r.Result.Error = err
	return r.Result, err
}

func (r *RTFetchValidator) ValidateResponse(ctx context.Context, atx tldb.Adapter, fn string, fr request.FetchResponse) (FetchValidationResult, error) {
	// Validate
	v := FetchValidationResult{}
	v.UploadTmpfile = fn
	v.UploadFilename = fmt.Sprintf("%s.pb", fr.ResponseSHA1)
	v.Found = false
	r.Result.Message, v.Error = rt.ReadFile(fn)
	return v, nil
}
