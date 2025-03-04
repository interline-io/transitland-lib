package fetch

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
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
	MaxSize                 uint64
	HideURL                 bool
	FetchedAt               time.Time
	Secrets                 []dmfr.Secret
	CreatedBy               tt.String
	Name                    tt.String
	Description             tt.String
	StrictValidation        bool
	SaveValidationReport    bool
	ValidationReportStorage string
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

type ValidationResult struct {
	UploadTmpfile  string
	UploadFilename string
	Error          error
	Found          bool
	FeedVersionID  tt.Int
}

type FetchValidator interface {
	ValidateResponse(context.Context, tldb.Adapter, request.FetchResponse, Options) (ValidationResult, error)
}

// Fetch and check for serious errors - regular errors are in fr.FetchError
func Fetch(ctx context.Context, atx tldb.Adapter, opts Options, cb FetchValidator) (Result, error) {
	result := Result{URL: opts.FeedURL}
	if cb == nil {
		return result, errors.New("no validator provided")
	}
	feed := dmfr.Feed{}
	if err := atx.Get(ctx, &feed, "select * from current_feeds where id = ?", opts.FeedID); err != nil {
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
	if opts.MaxSize > 0 {
		reqOpts = append(reqOpts, request.WithMaxSize(opts.MaxSize))
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
	fetchResponse, err := request.AuthenticatedRequestDownload(ctx, opts.FeedURL, reqOpts...)
	result.FetchError = fetchResponse.FetchError
	result.ResponseCode = fetchResponse.ResponseCode
	result.ResponseSize = fetchResponse.ResponseSize
	result.ResponseSHA1 = fetchResponse.ResponseSHA1
	result.ResponseTimeMs = fetchResponse.ResponseTimeMs
	result.ResponseTtfbMs = fetchResponse.ResponseTtfbMs
	if err != nil {
		return result, nil
	}

	// Fetch OK, validate
	newFile := false
	uploadFile := ""
	uploadDest := ""
	if result.FetchError == nil {
		vr, err := cb.ValidateResponse(ctx, atx, fetchResponse, opts)
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
		if vr.FeedVersionID.Valid {
			result.FeedVersionID.Set(vr.FeedVersionID.Val)
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
		store, err := request.GetStore(opts.Storage)
		if err != nil {
			return result, err
		}
		if err := request.Upload(ctx, store, uploadFile, uploadDest); err != nil {
			return result, err
		}
	}

	// Prepare and save feed fetch record
	tlfetch := dmfr.FeedFetch{}
	tlfetch.FeedID = feed.ID
	tlfetch.URLType = opts.URLType
	tlfetch.FetchedAt.Set(opts.FetchedAt)

	// Save response details, even if local filesystem
	if result.ResponseCode > 0 {
		tlfetch.ResponseCode.SetInt(result.ResponseCode)
	}
	tlfetch.ResponseSize.SetInt(result.ResponseSize)
	tlfetch.ResponseTimeMs.SetInt(result.ResponseTimeMs)
	tlfetch.ResponseTtfbMs.SetInt(result.ResponseTtfbMs)
	tlfetch.ResponseSHA1.Set(result.ResponseSHA1)

	// tlfetch.FeedVersionID =
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
	if _, err := atx.Insert(ctx, &tlfetch); err != nil {
		return result, err
	}
	return result, nil
}
