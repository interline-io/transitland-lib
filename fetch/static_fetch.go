package fetch

import (
	"archive/zip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/interline-io/transitland-lib/validator"
)

// StaticFetch from a URL. Creates FeedVersion and FeedFetch records.
// Returns an error if a serious failure occurs, such as database or filesystem access.
// Sets Result.FetchError if a regular failure occurs, such as a 404.
func StaticFetch(ctx context.Context, atx tldb.Adapter, opts StaticFetchOptions) (StaticFetchResult, error) {
	sfv := NewStaticFetchValidator(opts)
	return sfv.Fetch(ctx, atx)
}

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

type StaticFetchValidator struct {
	FeedVersion                *dmfr.FeedVersion
	FeedVersionValidatorResult *validator.Result
	StaticFetchOptions         StaticFetchOptions
}

func NewStaticFetchValidator(opts StaticFetchOptions) *StaticFetchValidator {
	return &StaticFetchValidator{StaticFetchOptions: opts}
}

func (sfv *StaticFetchValidator) Fetch(ctx context.Context, atx tldb.Adapter) (StaticFetchResult, error) {
	fetchResult, err := Fetch(ctx, atx, sfv.StaticFetchOptions.Options, sfv)
	if err != nil {
		log.For(ctx).Error().Err(err).Msg("fatal error during static fetch")
	}
	staticFetchResult := StaticFetchResult{
		Result:                     fetchResult,
		FeedVersion:                sfv.FeedVersion,
		FeedVersionValidatorResult: sfv.FeedVersionValidatorResult,
	}
	return staticFetchResult, err
}

func (sfv *StaticFetchValidator) ValidateResponse(ctx context.Context, atx tldb.Adapter, fn string, fetchResponse request.FetchResponse) (FetchValidationResult, error) {
	opts := sfv.StaticFetchOptions
	fetchValidationResult := FetchValidationResult{}

	// Open reader
	fragment := ""
	readerPath := fn
	if _, frag, ok := strings.Cut(opts.FeedURL, "#"); ok {
		readerPath = readerPath + "#" + frag
		fragment = frag
	}

	// Verify ZIP file integrity before validation
	if err := verifyZipIntegrity(readerPath); err != nil {
		fetchValidationResult.Error = fmt.Errorf("ZIP file integrity check failed: %w", err)
		return fetchValidationResult, nil
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

	// If a second tmpfile is created, copy it out since it will be deleted on reader.Close()
	fetchValidationResult.UploadTmpfile = reader.Path()
	fetchValidationResult.UploadFilename = fmt.Sprintf("%s.zip", fv.SHA1)
	if readerPath := reader.Path(); readerPath != fn {
		// Set fragment to empty
		fv.Fragment.Set("")
		// This file will be removed after upload
		uploadTmpfile, err := os.CreateTemp("", "fetch-nested")
		if err != nil {
			// Fatal error
			return fetchValidationResult, err
		}
		uploadTmpfile.Close() // close immediately
		fetchValidationResult.UploadTmpfile = uploadTmpfile.Name()
		// Copy file to file
		log.For(ctx).Info().Str("dst", fetchValidationResult.UploadTmpfile).Str("src", readerPath).Msg("fetch: copying extracted nested zip file for upload")
		if err := copyFileContents(fetchValidationResult.UploadTmpfile, readerPath); err != nil {
			// Fatal err
			return fetchValidationResult, err
		}
	}

	// Create a validation report
	validatorOptions := opts.ValidatorOptions
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

	// Generate feed version stats
	// TODO: Integrate this with static validation, so only one pass is necessary?
	feedVersionStats, err := stats.NewFeedStatsFromReader(reader)
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

// verifyZipFile verifies a ZIP file can be opened and all files can be read.
// This catches ZIP corruption early by attempting to read each file.
func verifyZipFile(path string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer r.Close()
	return verifyZipFiles(r.File)
}

// verifyZipIntegrity attempts to read all files in the ZIP archive to verify they're not corrupted.
// This catches ZIP corruption early, before expensive GTFS validation.
// For nested ZIPs (fragment ending in .zip), also verifies the nested ZIP's internal structure.
func verifyZipIntegrity(readerPath string) error {
	// Remove fragment if present
	path, fragment, _ := strings.Cut(readerPath, "#")

	// Open outer ZIP once and verify it
	r, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer r.Close()

	// Verify outer ZIP files can be read
	if err := verifyZipFiles(r.File); err != nil {
		return err
	}

	// If fragment points to a nested ZIP, verify it too
	if fragment != "" && strings.HasSuffix(fragment, ".zip") {
		// Find the nested ZIP file in the outer ZIP
		nestedZipIdx := slices.IndexFunc(r.File, func(f *zip.File) bool {
			return f.Name == fragment
		})
		if nestedZipIdx == -1 {
			return fmt.Errorf("nested ZIP file not found: %s", fragment)
		}
		nestedZipFile := r.File[nestedZipIdx]

		// Extract nested ZIP to temp file and verify it
		rc, err := nestedZipFile.Open()
		if err != nil {
			return fmt.Errorf("cannot open nested ZIP %s: %w", fragment, err)
		}
		defer rc.Close()

		tmpfile, err := os.CreateTemp("", "verify-nested-*.zip")
		if err != nil {
			return fmt.Errorf("cannot create temp file: %w", err)
		}
		defer os.Remove(tmpfile.Name())
		defer tmpfile.Close()

		if _, err := io.Copy(tmpfile, rc); err != nil {
			return fmt.Errorf("cannot extract nested ZIP %s: %w", fragment, err)
		}
		tmpfile.Close()

		// Verify nested ZIP using the same helper
		if err := verifyZipFile(tmpfile.Name()); err != nil {
			return fmt.Errorf("nested ZIP %s: %w", fragment, err)
		}
	}

	return nil
}

// verifyZipFiles verifies that all files in a ZIP archive can be opened and read.
func verifyZipFiles(files []*zip.File) error {
	for _, f := range files {
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("cannot open %s: %w", f.Name, err)
		}
		// Read a small amount to verify decompression works
		buf := make([]byte, 1024)
		_, err = rc.Read(buf)
		rc.Close()
		if err != nil && err != io.EOF {
			return fmt.Errorf("cannot read %s: %w", f.Name, err)
		}
	}
	return nil
}
