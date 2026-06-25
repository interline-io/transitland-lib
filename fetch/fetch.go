package fetch

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/feedmanager"
	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/tt"
)

// Options sets options for a fetch operation.
type Options struct {
	FeedURL                  string
	FeedID                   int
	URLType                  string
	Storage                  string
	AllowFTPFetch            bool
	AllowLocalFetch          bool
	AllowS3Fetch             bool
	AllowHTTPFetchUnfiltered bool
	MaxSize                  uint64
	HideURL                  bool
	FetchedAt                time.Time
	Secrets                  []dmfr.Secret
}

// Result contains results of a fetch operation.
type Result struct {
	Found          bool
	Error          error
	URL            string
	ResponseSize   int
	ResponseCode   int
	ResponseTtfbMs int
	ResponseTimeMs int
	ResponseSHA1   string
	FetchError     error
	FeedVersionID  tt.Int
}

// fetchDurations carries the validation/upload timings into the feed_fetch record.
type fetchDurations struct {
	validationMs int
	uploadMs     int
}

// download is the shared front half of every fetch: load the feed, apply auth,
// and download the URL to a tmpfile. The caller is responsible for removing the
// returned tmpfile. A non-nil error is fatal; a regular failure (no URL, bad
// secret, 404) is left on the returned response's FetchError.
func download(ctx context.Context, fm feedmanager.FeedManager, opts Options) (*dmfr.Feed, string, request.FetchResponse, error) {
	feed, err := fm.GetFeed(ctx, opts.FeedID)
	if err != nil {
		return nil, "", request.FetchResponse{}, err
	}
	if opts.FeedURL == "" {
		return feed, "", request.FetchResponse{FetchError: errors.New("no url provided")}, nil
	}
	var reqOpts []request.RequestOption
	if opts.AllowFTPFetch {
		reqOpts = append(reqOpts, request.WithAllowFTP)
	}
	if opts.AllowLocalFetch {
		reqOpts = append(reqOpts, request.WithAllowLocal)
	}
	if opts.AllowS3Fetch {
		reqOpts = append(reqOpts, request.WithAllowS3)
	}
	if opts.AllowHTTPFetchUnfiltered {
		reqOpts = append(reqOpts, request.WithAllowHTTPUnfiltered)
	}
	if opts.MaxSize > 0 {
		reqOpts = append(reqOpts, request.WithMaxSize(opts.MaxSize))
	}
	if feed.Authorization.Type != "" {
		secret, err := feed.MatchSecrets(opts.Secrets, opts.URLType)
		if err != nil {
			return feed, "", request.FetchResponse{FetchError: err}, nil
		}
		reqOpts = append(reqOpts, request.WithAuth(secret, feed.Authorization))
	}
	tmpfile, resp, fatal := request.AuthenticatedRequestDownload(ctx, opts.FeedURL, reqOpts...)
	return feed, tmpfile, resp, fatal
}

// resultFromResponse seeds a Result from the download response metadata.
func resultFromResponse(url string, resp request.FetchResponse) Result {
	return Result{
		URL:            url,
		FetchError:     resp.FetchError,
		ResponseCode:   resp.ResponseCode,
		ResponseSize:   resp.ResponseSize,
		ResponseSHA1:   resp.ResponseSHA1,
		ResponseTimeMs: resp.ResponseTimeMs,
		ResponseTtfbMs: resp.ResponseTtfbMs,
	}
}

// uploadFile stores a local file at key in the configured storage, returning the
// time spent. A no-op (0, nil) when storage, file, or key is empty.
func uploadFile(ctx context.Context, storage, fn, key string) (int, error) {
	if storage == "" || fn == "" || key == "" {
		return 0, nil
	}
	t := time.Now()
	store, err := request.GetStore(storage)
	if err != nil {
		return 0, err
	}
	if err := request.Upload(ctx, store, fn, key); err != nil {
		return 0, err
	}
	return int(time.Since(t).Milliseconds()), nil
}

// archiveKey builds the archive object key; the feed/url_type/date partitions let
// query engines prune by those columns.
func archiveKey(onestopID, urlType string, fetchedAt time.Time, ext string) string {
	t := fetchedAt.UTC()
	return fmt.Sprintf("feed=%s/url_type=%s/date=%s/%s.%s",
		onestopID, urlType, t.Format("2006-01-02"), t.Format("2006-01-02-15-04-05"), ext)
}

// recordFeedFetch writes the feed_fetch audit row for a completed attempt.
func recordFeedFetch(ctx context.Context, fm feedmanager.FeedManager, feed *dmfr.Feed, opts Options, result Result, dur fetchDurations, storageKey string) error {
	tlfetch := dmfr.FeedFetch{}
	tlfetch.FeedID = feed.ID
	tlfetch.URLType = opts.URLType
	tlfetch.FetchedAt.Set(opts.FetchedAt)
	if storageKey != "" {
		tlfetch.StorageKey.Set(storageKey)
	}
	if result.ResponseCode > 0 {
		tlfetch.ResponseCode.SetInt(result.ResponseCode)
	}
	tlfetch.ResponseSize.SetInt(result.ResponseSize)
	tlfetch.ResponseTimeMs.SetInt(result.ResponseTimeMs)
	tlfetch.ResponseTtfbMs.SetInt(result.ResponseTtfbMs)
	tlfetch.ResponseSHA1.Set(result.ResponseSHA1)
	tlfetch.ValidationDurationMs.SetInt(dur.validationMs)
	tlfetch.UploadDurationMs.SetInt(dur.uploadMs)
	if !opts.HideURL {
		tlfetch.URL = opts.FeedURL
	}
	if result.FeedVersionID.Valid {
		tlfetch.FeedVersionID.Set(result.FeedVersionID.Val)
	}
	if result.FetchError == nil {
		tlfetch.Success = true
	} else {
		tlfetch.Success = false
		tlfetch.FetchError.Set(result.FetchError.Error())
	}
	return fm.CreateFeedFetch(ctx, &tlfetch)
}
