package fetch

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/feedmanager"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/interline-io/transitland-lib/validator"
)

type StaticFetchResult struct {
	FeedVersion                *dmfr.FeedVersion
	FeedVersionValidatorResult *validator.Result
	Result
}

type StaticFetchOptions struct {
	StrictValidation        bool
	SaveValidationReport    bool
	ValidationReportStorage string
	ValidatorOptions        validator.Options
	CreatedBy               tt.String
	Name                    tt.String
	Description             tt.String
	Options
}

// StaticFetch downloads a GTFS feed, and — if it's new and valid — uploads the
// file and creates the FeedVersion + stats records. Creates a FeedFetch record
// either way. Returns an error only on a serious (db/filesystem) failure; a
// regular failure such as a 404 or strict-validation error is on Result.FetchError.
func StaticFetch(ctx context.Context, fm feedmanager.FeedManager, opts StaticFetchOptions) (StaticFetchResult, error) {
	out := StaticFetchResult{}
	feed, tmpfile, resp, fatal := download(ctx, fm, opts.Options)
	if tmpfile != "" {
		defer os.Remove(tmpfile)
	}
	out.Result = resultFromResponse(opts.FeedURL, resp)
	if fatal != nil {
		log.For(ctx).Error().Err(fatal).Msg("fatal error during static fetch")
		return out, fatal
	}

	var dur fetchDurations
	if out.FetchError == nil {
		if err := staticProcess(ctx, fm, tmpfile, opts, &out, &dur); err != nil {
			log.For(ctx).Error().Err(err).Msg("fatal error during static fetch")
			return out, err
		}
	}
	// Static uploads are content-addressed (<sha1>.zip), so no storage_key.
	if err := recordFeedFetch(ctx, fm, feed, opts.Options, out.Result, dur, ""); err != nil {
		return out, err
	}
	return out, nil
}

// staticProcess validates the downloaded feed and, if it's new and valid, uploads
// the file and writes the feed version + stats + report in one transaction. It
// updates out (Found / FeedVersionID / FetchError / FeedVersion / validator
// result) and returns only fatal errors. The upload happens before the writes, so
// a rolled-back write can never leave a feed_version referencing a missing file.
func staticProcess(ctx context.Context, fm feedmanager.FeedManager, fn string, opts StaticFetchOptions, out *StaticFetchResult, dur *fetchDurations) error {
	// Open reader (handle a #fragment selecting a feed nested in the archive).
	fragment := ""
	readerPath := fn
	if _, frag, ok := strings.Cut(opts.FeedURL, "#"); ok {
		readerPath = readerPath + "#" + frag
		fragment = frag
	}
	reader, err := tlcsv.NewReaderFromAdapter(tlcsv.NewZipAdapter(readerPath))
	if err != nil {
		out.FetchError = err
		return nil
	}
	if err := reader.Open(); err != nil {
		out.FetchError = err
		return nil
	}
	defer reader.Close()

	// Build the feed version (sha1, service dates, structure check).
	fv, err := stats.NewFeedVersionFromReader(reader)
	if err != nil {
		out.FetchError = err
		return nil
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

	// Skip the work if we already have this feed version.
	if checkFv, err := fm.GetFeedVersionBySHA1(ctx, fv.SHA1, fv.SHA1Dir.Val); err != nil {
		return err
	} else if checkFv != nil {
		out.Found = true
		out.FeedVersionID.SetInt(checkFv.ID)
		out.FeedVersion = checkFv
		return nil
	}

	// Determine the file to upload. A nested feed is extracted to its own tmpfile
	// since reader.Path() is removed on reader.Close().
	uploadFn := reader.Path()
	uploadKey := fmt.Sprintf("%s.zip", fv.SHA1)
	if uploadFn != fn {
		fv.Fragment.Set("")
		nested, err := os.CreateTemp("", "fetch-nested")
		if err != nil {
			return err
		}
		nested.Close()
		defer os.Remove(nested.Name())
		log.For(ctx).Info().Str("dst", nested.Name()).Str("src", uploadFn).Msg("fetch: copying extracted nested zip file for upload")
		if err := copyFileContents(nested.Name(), uploadFn); err != nil {
			return err
		}
		uploadFn = nested.Name()
	}

	// Validate + compute stats.
	validationStart := time.Now()
	validatorOptions := opts.ValidatorOptions
	validatorOptions.ErrorLimit = 10
	v, err := validator.NewValidator(reader, validatorOptions)
	if err != nil {
		return err
	}
	validationResult, err := v.Validate(ctx)
	if err != nil {
		return err
	}
	// TODO: integrate this with validation, so only one pass is necessary?
	feedVersionStats, err := stats.NewFeedStatsFromReader(reader)
	if err != nil {
		return err
	}
	dur.validationMs = int(time.Since(validationStart).Milliseconds())

	// Strict validation: reject without uploading or writing anything.
	if opts.StrictValidation && len(validationResult.Errors) > 0 {
		out.FetchError = fmt.Errorf("strict validation failed, errors in %d files", len(validationResult.Errors))
		return nil
	}

	// Upload BEFORE the database writes, so a committed feed_version can never
	// reference a file that failed to upload (stats include the stop geohash cells
	// for feed/feed_version spatial queries).
	uploadMs, err := uploadFile(ctx, opts.Storage, uploadFn, uploadKey)
	if err != nil {
		return err
	}
	dur.uploadMs = uploadMs

	// Re-check for a feed version a concurrent fetch committed during validation +
	// upload, to avoid the fatal unique-index violation on sha1 where possible,
	// then write feed version + stats + report in one transaction.
	if checkFv, err := fm.GetFeedVersionBySHA1(ctx, fv.SHA1, fv.SHA1Dir.Val); err != nil {
		return err
	} else if checkFv != nil {
		out.Found = true
		out.FeedVersionID.SetInt(checkFv.ID)
		out.FeedVersion = checkFv
		return nil
	}
	if err := fm.WithTx(ctx, func(ctx context.Context, tx feedmanager.FeedManager) error {
		if _, err := tx.CreateFeedVersion(ctx, &fv); err != nil {
			return err
		}
		if err := tx.WriteFeedVersionStats(ctx, fv.ID, feedVersionStats); err != nil {
			return err
		}
		if opts.ValidationReportStorage != "" {
			return tx.SaveValidationReport(ctx, fv.ID, validationResult, opts.ValidationReportStorage)
		}
		return nil
	}); err != nil {
		return err
	}

	out.FeedVersionID.SetInt(fv.ID)
	out.FeedVersion = &fv
	out.FeedVersionValidatorResult = validationResult
	return nil
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
