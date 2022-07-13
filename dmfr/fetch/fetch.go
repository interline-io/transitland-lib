package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/log"
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
	Directory               string
	S3                      string
	AllowS3Fetch            bool
	AllowFTPFetch           bool
	AllowLocalFetch         bool
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
		secret, err := feed.MatchSecrets(opts.Secrets)
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
	if newFile && uploadFile != "" {
		if opts.Directory != "" {
			outfn := filepath.Join(opts.Directory, uploadDest)
			log.Debug().Str("src", uploadFile).Str("dst", outfn).Msg("fetch: copying file to gtfs dir")
			if err := copyFileContents(outfn, uploadFile); err != nil {
				return result, err
			}
		}
		if opts.S3 != "" {
			ustr := fmt.Sprintf("%s/%s", opts.S3, uploadDest)
			log.Debug().Str("src", uploadFile).Str("dst", ustr).Msg("fetch: copying file to s3")
			rp, err := os.Open(uploadFile)
			if err != nil {
				return result, err
			}
			defer rp.Close()
			if err := request.UploadS3(context.Background(), ustr, tl.Secret{}, rp); err != nil {
				return result, err
			}
		}
	}

	// Prepare and save feed fetch record
	tlfetch := dmfr.FeedFetch{}
	tlfetch.FeedID = feed.ID
	tlfetch.URLType = opts.URLType
	tlfetch.URL = opts.FeedURL
	tlfetch.FetchedAt = tt.NewTime(opts.FetchedAt)
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
	tlfetch.UpdateTimestamps()
	if _, err := atx.Insert(&tlfetch); err != nil {
		return result, err
	}
	return result, nil
}

func copyFileContents(dst, src string) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}
