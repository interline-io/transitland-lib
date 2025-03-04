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
	FeedVersion      *dmfr.FeedVersion
	ValidationResult *validator.Result
	Result
}

// StaticFetch from a URL. Creates FeedVersion and FeedFetch records.
// Returns an error if a serious failure occurs, such as database or filesystem access.
// Sets Result.FetchError if a regular failure occurs, such as a 404.
// feed is an argument to provide the ID, File, and Authorization.
func StaticFetch(ctx context.Context, atx tldb.Adapter, opts Options) (StaticFetchResult, error) {
	cb := &staticFetchValidator{}
	result, err := Fetch(ctx, atx, opts, cb)
	if err != nil {
		log.For(ctx).Error().Err(err).Msg("fatal error during static fetch")
	}
	cb.ret.Result = result
	cb.ret.Error = err
	return cb.ret, err
}

type staticFetchValidator struct {
	ret StaticFetchResult
}

func (r *staticFetchValidator) ValidateResponse(ctx context.Context, atx tldb.Adapter, fr request.FetchResponse, opts Options) (validationResponse, error) {
	tmpfilepath := fr.Filename
	vr := validationResponse{}

	// Open reader
	fragment := ""
	if a := strings.SplitN(opts.FeedURL, "#", 2); len(a) > 1 {
		tmpfilepath = tmpfilepath + "#" + a[1]
		fragment = a[1]
	}
	reader, err := tlcsv.NewReaderFromAdapter(tlcsv.NewZipAdapter(tmpfilepath))
	if err != nil {
		vr.Error = err
		return vr, nil
	}
	if err := reader.Open(); err != nil {
		vr.Error = err
		return vr, nil
	}
	defer reader.Close()

	// Get initialized FeedVersion
	fv, err := stats.NewFeedVersionFromReader(reader)
	if err != nil {
		vr.Error = err
		return vr, nil
	}
	r.ret.FeedVersion = &fv
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

	// Is this SHA1 already present?
	checkfvid := dmfr.FeedVersion{}
	err = atx.Get(ctx, &checkfvid, "SELECT * FROM feed_versions WHERE sha1 = ? OR sha1_dir = ? LIMIT 1", fv.SHA1, fv.SHA1Dir)
	if err == nil {
		// Already present
		fv = checkfvid
		vr.Found = true
		vr.FeedVersionID.SetInt(fv.ID)
		return vr, nil
	} else if err == sql.ErrNoRows {
		// Not present, create below
	} else {
		// Serious error
		return vr, err
	}

	// If a second tmpfile is created, copy it and overwrite the input tmp file
	vr.UploadTmpfile = reader.Path()
	vr.UploadFilename = fv.File
	if readerPath := reader.Path(); readerPath != fr.Filename {
		// Set fragment to empty
		fv.Fragment.Set("")
		// Copy file
		tf2, err := os.CreateTemp("", "nested")
		if err != nil {
			return vr, err
		}
		vr.UploadTmpfile = tf2.Name()
		tf2.Close()
		log.For(ctx).Info().Str("dst", vr.UploadTmpfile).Str("src", readerPath).Msg("fetch: copying extracted nested zip file for upload")
		if err := copyFileContents(vr.UploadTmpfile, readerPath); err != nil {
			// Fatal err
			return vr, err
		}
	}

	// Create fv record
	fv.ID, err = atx.Insert(ctx, &fv)
	if err != nil {
		// Fatal err
		return vr, err
	}
	vr.FeedVersionID.SetInt(fv.ID)

	// Create a validation report
	if opts.SaveValidationReport || opts.StrictValidation {
		// Create new report
		validatorOptions := validator.Options{}
		validatorOptions.ErrorLimit = 10
		v, err := validator.NewValidator(reader, validatorOptions)
		if err != nil {
			return vr, err
		}
		validationResult, err := v.Validate(ctx)
		if err != nil {
			return vr, err
		}
		r.ret.ValidationResult = validationResult

		// Strict validation; fail if errors
		errCount := len(validationResult.Errors)
		if opts.StrictValidation && errCount > 0 {
			return vr, fmt.Errorf("strict validation failed with %d errors", errCount)
		}

		// Save validation report
		if opts.ValidationReportStorage != "" {
			if err := validator.SaveValidationReport(ctx, atx, validationResult, fv.ID, opts.ValidationReportStorage); err != nil {
				return vr, err
			}
		}
	}

	// Update stats records
	if err := stats.CreateFeedStats(ctx, atx, reader, fv.ID); err != nil {
		// Fatal err
		return vr, err
	}
	return vr, nil
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
