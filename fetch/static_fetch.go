package fetch

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tlutil"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/interline-io/transitland-lib/validator"
)

type StaticFetchResult struct {
	FeedVersion      *tl.FeedVersion
	ValidationResult *validator.Result
	Result
}

// StaticFetch from a URL. Creates FeedVersion and FeedFetch records.
// Returns an error if a serious failure occurs, such as database or filesystem access.
// Sets Result.FetchError if a regular failure occurs, such as a 404.
// feed is an argument to provide the ID, File, and Authorization.
func StaticFetch(atx tldb.Adapter, opts Options) (StaticFetchResult, error) {
	var ret StaticFetchResult
	cb := func(fr request.FetchResponse) (validationResponse, error) {
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
		fv, err := tlutil.NewFeedVersionFromReader(reader)
		if err != nil {
			vr.Error = err
			return vr, nil
		}
		ret.FeedVersion = &fv
		fv.FeedID = opts.FeedID
		fv.FetchedAt = opts.FetchedAt
		fv.CreatedBy = opts.CreatedBy
		fv.Name = opts.Name
		fv.Description = opts.Description
		fv.File = fmt.Sprintf("%s.zip", fv.SHA1)
		fv.Fragment = tt.NewString(fragment)
		if !opts.HideURL {
			fv.URL = opts.FeedURL
		}

		// Is this SHA1 already present?
		checkfvid := tl.FeedVersion{}
		err = atx.Get(&checkfvid, "SELECT * FROM feed_versions WHERE sha1 = ? OR sha1_dir = ? LIMIT 1", fv.SHA1, fv.SHA1Dir)
		if err == nil {
			// Already present
			fv = checkfvid
			vr.Found = true
			vr.FeedVersionID = tt.NewInt(fv.ID)
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
			fv.Fragment = tt.NewString("")
			// Copy file
			tf2, err := os.CreateTemp("", "nested")
			if err != nil {
				return vr, err
			}
			vr.UploadTmpfile = tf2.Name()
			tf2.Close()
			log.Info().Str("dst", vr.UploadTmpfile).Str("src", readerPath).Msg("fetch: copying extracted nested zip file for upload")
			if err := copyFileContents(vr.UploadTmpfile, readerPath); err != nil {
				// Fatal err
				return vr, err
			}
		}

		// Create fv record
		fv.ID, err = atx.Insert(&fv)
		if err != nil {
			// Fatal err
			return vr, err
		}
		vr.FeedVersionID = tt.NewInt(fv.ID)

		// Update validation report
		if opts.SaveValidationReport {
			validationResult, err := createFeedValidationReport(atx, reader, fv.ID, opts.FetchedAt, opts.ValidationReportStorage)
			if err != nil {
				// Fatal err
				return vr, err
			}
			ret.ValidationResult = validationResult
		}

		// Update stats records
		if err := stats.CreateFeedStats(atx, reader, fv.ID); err != nil {
			// Fatal err
			return vr, err
		}
		return vr, nil
	}
	result, err := ffetch(atx, opts, cb)
	if err != nil {
		log.Error().Err(err).Msg("fatal error during static fetch")
	}
	ret.Result = result
	ret.Error = err
	return ret, err
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

// Duplicated
func createFeedValidationReport(atx tldb.Adapter, reader *tlcsv.Reader, fvid int, fetchedAt time.Time, storage string) (*validator.Result, error) {
	// Create new report
	_ = fetchedAt
	opts := validator.Options{}
	opts.ErrorLimit = 10
	v, err := validator.NewValidator(reader, opts)
	if err != nil {
		return nil, err
	}
	validationResult, err := v.Validate()
	if err != nil {
		return nil, err
	}
	if err := validator.SaveValidationReport(atx, validationResult, fvid, storage); err != nil {
		return nil, err
	}
	return validationResult, nil
}
