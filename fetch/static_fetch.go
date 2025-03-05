package fetch

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/validator"
)

type StaticFetchResult struct {
	FeedVersion                *dmfr.FeedVersion
	FeedVersionValidatorResult *validator.Result
	Result
}

// StaticFetch from a URL. Creates FeedVersion and FeedFetch records.
// Returns an error if a serious failure occurs, such as database or filesystem access.
// Sets Result.FetchError if a regular failure occurs, such as a 404.
func StaticFetch(ctx context.Context, atx tldb.Adapter, opts Options) (StaticFetchResult, error) {
	cb := &StaticFetchValidator{}
	fetchResult, err := Fetch(ctx, atx, opts, cb)
	if err != nil {
		log.For(ctx).Error().Err(err).Msg("fatal error during static fetch")
	}
	staticFetchResult := StaticFetchResult{
		Result:                     fetchResult,
		FeedVersion:                cb.FeedVersion,
		FeedVersionValidatorResult: cb.FeedVersionValidatorResult,
	}
	return staticFetchResult, err
}

type StaticFetchValidator struct {
	FeedVersion                *dmfr.FeedVersion
	FeedVersionValidatorResult *validator.Result
}

func (sfv *StaticFetchValidator) ValidateResponse(ctx context.Context, atx tldb.Adapter, fn string, fetchResponse request.FetchResponse, opts Options) (FetchValidationResult, error) {
	fetchValidationResult := FetchValidationResult{}

	// Open reader
	fragment := ""
	readerPath := fn
	if a := strings.SplitN(opts.FeedURL, "#", 2); len(a) > 1 {
		readerPath = readerPath + "#" + a[1]
		fragment = a[1]
	}
	reader, err := tlcsv.NewReaderFromAdapter(tlcsv.NewZipAdapter(readerPath))
	if err != nil {
		fetchValidationResult.Error = err
		return fetchValidationResult, nil
	}
	if err := reader.Open(); err != nil {
		fetchValidationResult.Error = err
		return fetchValidationResult, nil
	}
	defer reader.Close()

	// Get initialized FeedVersion
	fv, err := stats.NewFeedVersionFromReader(reader)
	if err != nil {
		fetchValidationResult.Error = err
		return fetchValidationResult, nil
	}
	fv.FeedID = opts.FeedID
	fv.FetchedAt = opts.FetchedAt
	fv.CreatedBy = opts.CreatedBy
	fv.Name = opts.Name
	fv.Description = opts.Description
	fv.File = fmt.Sprintf("%s.zip", fv.SHA1)
	fv.Fragment.Set(fragment)
	if !opts.HideURL {
		fv.URL = opts.FeedURL
	}

	// If the fv is already present, return.
	// This is to skip unncessary work, not avoid duplicates. A second check is done later.
	if checkFv, err := checkFeedVersion(ctx, atx, fv.SHA1, fv.SHA1Dir.Val); err != nil {
		// Fatal error
		return fetchValidationResult, err
	} else if checkFv != nil {
		// Already present
		fetchValidationResult.Found = true
		fetchValidationResult.FeedVersionID.SetInt(checkFv.ID)
		sfv.FeedVersion = checkFv
		return fetchValidationResult, nil
	}

	// If a second tmpfile is created, copy it and overwrite the input tmp file
	fetchValidationResult.UploadTmpfile = reader.Path()
	fetchValidationResult.UploadFilename = fmt.Sprintf("%s.zip", fv.SHA1)
	if readerPath := reader.Path(); readerPath != fn {
		// Set fragment to empty
		fv.Fragment.Set("")
		// This file will be removed after upload
		uploadTmpfile, err := os.CreateTemp("", "nested")
		if err != nil {
			// Fatal error
			return fetchValidationResult, err
		}
		uploadTmpfile.Close() // close immediately
		fetchValidationResult.UploadTmpfile = uploadTmpfile.Name()
		log.For(ctx).Info().Str("dst", fetchValidationResult.UploadTmpfile).Str("src", readerPath).Msg("fetch: copying extracted nested zip file for upload")
		// Copy file to file
		if err := copyFileContents(fetchValidationResult.UploadTmpfile, readerPath); err != nil {
			// Fatal err
			return fetchValidationResult, err
		}
	}

	// Generate feed version stats
	feedVersionStats, err := stats.NewFeedStatsFromReader(reader)
	if err != nil {
		// Fatal error
		return fetchValidationResult, err
	}

	// Create a validation report
	validatorOptions := validator.Options{}
	validatorOptions.ErrorLimit = 10
	v, err := validator.NewValidator(reader, validatorOptions)
	if err != nil {
		// Fatal error
		return fetchValidationResult, err
	}
	validationResult, err := v.Validate(ctx)
	if err != nil {
		// Fatal error
		return fetchValidationResult, err
	}

	// Strict validation; do not save feed version
	errCount := len(validationResult.Errors)
	if opts.StrictValidation && errCount > 0 {
		fetchValidationResult.Error = fmt.Errorf("strict validation failed, errors in %d files", errCount)
		return fetchValidationResult, nil
	}

	// The validation after the initial check may take some time to complete, so check again.
	// We want to avoid database write failures (unique index on sha1) because those are considered fatal.
	if checkFv, err := checkFeedVersion(ctx, atx, fv.SHA1, fv.SHA1Dir.Val); err != nil {
		// Fatal error
		return fetchValidationResult, err
	} else if checkFv != nil {
		// Already present
		fetchValidationResult.Found = true
		fetchValidationResult.FeedVersionID.SetInt(checkFv.ID)
		sfv.FeedVersion = checkFv
		return fetchValidationResult, nil
	}

	// Create fv record
	if _, err = atx.Insert(ctx, &fv); err != nil {
		// Fatal err
		return fetchValidationResult, err
	}

	// Save stats records
	if err := stats.WriteFeedVersionStats(ctx, atx, feedVersionStats, fv.ID); err != nil {
		// Fatal err
		return fetchValidationResult, err
	}

	// Save validation report
	if opts.ValidationReportStorage != "" {
		if err := validator.SaveValidationReport(ctx, atx, validationResult, fv.ID, opts.ValidationReportStorage); err != nil {
			// Fatal error
			return fetchValidationResult, err
		}
	}

	// OK
	fetchValidationResult.Found = false
	fetchValidationResult.FeedVersionID.SetInt(fv.ID)
	sfv.FeedVersionValidatorResult = validationResult
	sfv.FeedVersion = &fv
	return fetchValidationResult, nil
}

// Is this SHA1 already present?
func checkFeedVersion(ctx context.Context, atx tldb.Adapter, sha1 string, sha1dir string) (*dmfr.FeedVersion, error) {
	checkFeedVersion := dmfr.FeedVersion{}
	err := atx.Get(ctx, &checkFeedVersion, "SELECT * FROM feed_versions WHERE sha1 = ? OR sha1_dir = ? LIMIT 1", sha1, sha1dir)
	if err == nil {
		return &checkFeedVersion, nil
	} else if err == sql.ErrNoRows {
		// Not present, create below
	} else {
		// Fatal error
		return nil, err
	}
	return nil, nil
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
