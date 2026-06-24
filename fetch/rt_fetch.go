package fetch

import (
	"context"
	"fmt"
	"os"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/feedmanager"
	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/rt/pb"
)

type RTFetchResult struct {
	Message *pb.FeedMessage
	Result
}

type RTFetchOptions struct {
	Options
}

// RTFetch downloads a GTFS-RT feed, parses the protobuf, uploads it, and records
// a FeedFetch. It creates no feed version. Returns an error only on a serious
// failure; a 404 or parse error is on Result.FetchError.
func RTFetch(ctx context.Context, fm feedmanager.FeedManager, opts RTFetchOptions) (RTFetchResult, error) {
	out := RTFetchResult{}
	feed, tmpfile, resp, fatal := download(ctx, fm, opts.Options)
	if tmpfile != "" {
		defer os.Remove(tmpfile)
	}
	out.Result = resultFromResponse(opts.FeedURL, resp)
	if fatal != nil {
		log.For(ctx).Error().Err(fatal).Msg("fatal error during rt fetch")
		return out, fatal
	}

	var dur fetchDurations
	if out.FetchError == nil {
		msg, err := rt.ReadFile(tmpfile)
		out.Message = msg
		if err != nil {
			out.FetchError = err
		}
		// Upload the protobuf (content-addressed key); matches the prior behavior
		// of uploading even when parsing failed.
		uploadMs, uerr := uploadFile(ctx, opts.Storage, tmpfile, fmt.Sprintf("%s.pb", resp.ResponseSHA1))
		if uerr != nil {
			log.For(ctx).Error().Err(uerr).Msg("fatal error during rt fetch")
			return out, uerr
		}
		dur.uploadMs = uploadMs
	}
	if err := recordFeedFetch(ctx, fm, feed, opts.Options, out.Result, dur); err != nil {
		return out, err
	}
	return out, nil
}
