package fetch

import (
	"errors"
	"fmt"
	"math"
	"os"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/dmfr/store"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/request"
	"github.com/interline-io/transitland-lib/tl/tt"
	"github.com/interline-io/transitland-lib/tldb"
)

// Options sets options for a fetch operation.
type Options struct {
	FeedURL                 string
	FeedID                  int
	URLType                 string
	IgnoreDuplicateContents bool
	Storage                 string
	AllowFTPFetch           bool
	AllowLocalFetch         bool
	AllowS3Fetch            bool
	HideURL                 bool
	FetchedAt               time.Time
	Secrets                 []tl.Secret
	CreatedBy               tl.String
	Name                    tl.String
	Description             tl.String
}

// Result contains results of a fetch operation.
type Result struct {
	Found        bool
	Error        error
	ResponseSize int
	ResponseCode int
	ResponseSHA1 string
	FetchError   error
}

type validationResponse struct {
	UploadTmpfile  string
	UploadFilename string
	Error          error
	Found          bool
}

type fetchCb func(request.FetchResponse) (validationResponse, error)

// Fetch and check for serious errors - regular errors are in fr.FetchError
func ffetch(atx tldb.Adapter, opts Options, cb fetchCb) (Result, error) {
	result := Result{}
	feed := tl.Feed{}
	if err := atx.Get(&feed, "select * from current_feeds where id = ?", opts.FeedID); err != nil {
		return result, err
	}
	if opts.FeedURL == "" {
		result.FetchError = errors.New("no url provided")
		return result, nil
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
	// Get secret and set auth
	if feed.Authorization.Type != "" {
		secret, err := feed.MatchSecrets(opts.Secrets, opts.URLType)
		if err != nil {
			result.FetchError = err
			return result, nil
		}
		reqOpts = append(reqOpts, request.WithAuth(secret, feed.Authorization))
	}
	fetchResponse, err := request.AuthenticatedRequestDownload(opts.FeedURL, reqOpts...)
	result.FetchError = fetchResponse.FetchError
	result.ResponseCode = fetchResponse.ResponseCode
	result.ResponseSize = fetchResponse.ResponseSize
	result.ResponseSHA1 = fetchResponse.ResponseSHA1
	if err != nil {
		return result, nil
	}

	// Fetch OK, validate
	newFile := false
	uploadFile := ""
	uploadDest := ""
	if result.FetchError == nil {
		vr, err := cb(fetchResponse)
		if err != nil {
			return result, err
		}
		result.FetchError = vr.Error
		result.Found = vr.Found
		if !result.Found {
			newFile = true
			uploadFile = vr.UploadTmpfile
			uploadDest = vr.UploadFilename
		}
	}

	// Cleanup any temporary files
	if fetchResponse.Filename != "" {
		defer os.Remove(fetchResponse.Filename)
	}
	if uploadFile != "" && uploadFile != fetchResponse.Filename {
		defer os.Remove(uploadFile)
	}

	// Validate OK, upload
	if newFile && uploadFile != "" && opts.Storage != "" {
		if err := store.UploadFile(opts.Storage, uploadFile, uploadDest); err != nil {
			return result, err
		}
	}

	// Prepare and save feed fetch record
	tlfetch := dmfr.FeedFetch{}
	tlfetch.FeedID = feed.ID
	tlfetch.URLType = opts.URLType
	tlfetch.FetchedAt = tt.NewTime(opts.FetchedAt)
	if !opts.HideURL {
		tlfetch.URL = opts.FeedURL
	}
	if result.ResponseCode > 0 {
		tlfetch.ResponseCode = tt.NewInt(result.ResponseCode)
		tlfetch.ResponseSize = tt.NewInt(result.ResponseSize)
		tlfetch.ResponseSHA1 = tt.NewString(result.ResponseSHA1)
	}
	if result.FetchError == nil {
		tlfetch.Success = true
	} else {
		tlfetch.Success = false
		tlfetch.FetchError = tt.NewString(result.FetchError.Error())
	}
	if _, err := atx.Insert(&tlfetch); err != nil {
		return result, err
	}
	return result, nil
}

func CheckFetchWait(atx tldb.Adapter, feedId int, fetchWait float64) (bool, error) {
	fmt.Println("CheckFetchWait:", feedId, "fetchWait:", fetchWait)
	// Check if minimum fetch time has elapsed
	now := time.Now()
	q := atx.Sqrl().
		Select("feed_fetches.*").
		From("feed_fetches").
		Where(sq.Eq{"feed_id": feedId}).
		OrderBy("fetched_at desc").
		Limit(10)

	var lastFetches []dmfr.FeedFetch
	if qstr, qargs, err := q.ToSql(); err != nil {
		return false, err
	} else if err := atx.Select(&lastFetches, qstr, qargs...); err != nil {
		return false, err
	}

	// Get exponential backoff
	// Based on failures in the past 24 hours
	failures := 0
	lastFetchAgo := 0.0
	checkFailureWindow := 60.0 * 60.0 * 24
	for _, fetch := range lastFetches {
		timeAgo := now.Sub(fetch.FetchedAt.Val).Seconds()
		fmt.Println("\tfetch:", fetch.ID, fetch.Success, fetch.FetchedAt.String(), "timeAgo:", timeAgo)
		if lastFetchAgo == 0 {
			lastFetchAgo = timeAgo
		}
		if fetch.Success {
			fmt.Println("\t\tbreaking on succes")
			break
		}
		if timeAgo > checkFailureWindow {
			fmt.Println("\t\tbreaking on time")
			break
		}
		failures += 1
	}
	fmt.Println("\tlastFetchAgo:", lastFetchAgo)

	failureBackoff := math.Pow(2, float64(failures)*2)
	fmt.Println("\tfailures:", failures, "failureBackoff:", failureBackoff)
	if failureBackoff > checkFailureWindow {
		failureBackoff = checkFailureWindow
	}
	if failures > 0 && lastFetchAgo < failureBackoff {
		fmt.Println("\tskipping, failure backoff:", failureBackoff)
		return false, nil
	}
	if lastFetchAgo < float64(fetchWait) {
		fmt.Println("\tskipping, fetch wait:", fetchWait)
		return false, nil
	}
	return true, nil
}
