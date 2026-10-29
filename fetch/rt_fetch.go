package fetch

import (
	"context"
	"os"
	"time"

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
	// Spool, when set, receives a copy of every downloaded message, partitioned
	// by URL type and hour: the input to batch jobs that read a whole hour of
	// every feed. Unlike Storage, the per-feed archive recorded on the
	// feed_fetch row, it applies to every feed, and a failed write is logged
	// rather than failing the fetch.
	Spool string
}

// RTFetch downloads a GTFS-RT feed, parses the protobuf, uploads it, and records
// a FeedFetch. It creates no feed version. Returns an error only on a serious
// failure; a 404 or parse error is on Result.FetchError.
func RTFetch(ctx context.Context, fm feedmanager.FeedManager, opts RTFetchOptions) (RTFetchResult, error) {
	out := RTFetchResult{}
	if opts.FetchedAt.IsZero() {
		opts.FetchedAt = time.Now().UTC()
	}
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
	var storageKey string
	if out.FetchError == nil {
		// The protobuf parse is the RT analogue of validation; record its duration
		// on the feed_fetch row, as the pre-rewrite fetch did.
		validationStart := time.Now()
		msg, err := rt.ReadFile(tmpfile)
		dur.validationMs = int(time.Since(validationStart).Milliseconds())
		out.Message = msg
		if err != nil {
			out.FetchError = err
		}
		// Archive to a partitioned key when storage is set, even on parse failure.
		if opts.Storage != "" {
			storageKey = archiveKey(feed.FeedID, opts.URLType, resp.ResponseSHA1, opts.FetchedAt, "pb")
		}
		uploadMs, uerr := uploadFile(ctx, opts.Storage, tmpfile, storageKey)
		if uerr != nil {
			log.For(ctx).Error().Err(uerr).Msg("fatal error during rt fetch")
			return out, uerr
		}
		dur.uploadMs = uploadMs
		// Spooled even on parse failure, like the archive, so a batch job sees
		// every message that arrived.
		if opts.Spool != "" {
			key := spoolKey(feed.FeedID, opts.URLType, resp.ResponseSHA1, opts.FetchedAt)
			if _, err := uploadFile(ctx, opts.Spool, tmpfile, key); err != nil {
				log.For(ctx).Error().Err(err).Str("key", key).Msg("rt fetch: spool write failed")
			}
		}
	}
	if err := recordFeedFetch(ctx, fm, feed, opts.Options, out.Result, dur, storageKey); err != nil {
		return out, err
	}
	return out, nil
}
