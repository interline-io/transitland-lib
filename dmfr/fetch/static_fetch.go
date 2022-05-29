package fetch

import (
	"database/sql"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/request"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
)

// StaticFetch from a URL. Creates FeedVersion and FeedFetch records.
// Returns an error if a serious failure occurs, such as database or filesystem access.
// Sets Result.FetchError if a regular failure occurs, such as a 404.
// feed is an argument to provide the ID, File, and Authorization.
func StaticFetch(atx tldb.Adapter, opts Options) (tl.FeedVersion, Result, error) {
	var fv tl.FeedVersion
	var fr Result

	// Get feed, create if not present and FeedCreate is specified
	feed := tl.Feed{}
	if err := atx.Get(&feed, `SELECT * FROM current_feeds WHERE onestop_id = ?`, opts.FeedID); err == sql.ErrNoRows && opts.FeedCreate {
		feed.FeedID = opts.FeedID
		feed.Spec = "gtfs"
		if feed.ID, err = atx.Insert(&feed); err != nil {
			return fv, fr, err
		}
	} else if err != nil {
		return fv, fr, errors.New("feed does not exist")
	}
	opts.URLType = "manual"
	if opts.FeedURL == "" {
		opts.URLType = "static_current"
		opts.FeedURL = feed.URLs.StaticCurrent
	}

	cb := func(fr request.FetchResponse) (validationResponse, error) {
		tmpfilepath := fr.Filename
		vr := validationResponse{}
		// Open reader
		if a := strings.SplitN(opts.FeedURL, "#", 2); len(a) > 1 {
			tmpfilepath = tmpfilepath + "#" + a[1]
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
		fv, err = tl.NewFeedVersionFromReader(reader)
		if err != nil {
			vr.Error = err
			return vr, nil
		}
		fv.URL = opts.FeedURL
		fv.FeedID = feed.ID
		fv.FetchedAt = opts.FetchedAt
		fv.CreatedBy = opts.CreatedBy
		fv.Name = opts.Name
		fv.Description = opts.Description
		fv.File = fmt.Sprintf("%s.zip", fv.SHA1)
		// Is this SHA1 already present?
		checkfvid := tl.FeedVersion{}
		err = atx.Get(&checkfvid, "SELECT * FROM feed_versions WHERE sha1 = ? OR sha1_dir = ?", fv.SHA1, fv.SHA1Dir)
		if err == nil {
			// Already present
			fv = checkfvid
			vr.Found = true
			return vr, nil
		} else if err == sql.ErrNoRows {
			// Not present, create below
		} else if err != nil {
			// Serious error
			return vr, err
		}
		// Return fv
		fv.UpdateTimestamps()
		fv.ID, err = atx.Insert(&fv)
		if err != nil {
			return vr, err
		}
		// Update stats records
		if err := createFeedStats(atx, reader, fv.ID); err != nil {
			return vr, err
		}
		// If a second tmpfile is created, copy it and overwrite the input tmp file
		vr.UploadTmpfile = reader.Path()
		vr.UploadFilename = fv.File
		if readerPath := reader.Path(); readerPath != fr.Filename {
			tf2, err := ioutil.TempFile("", "nested")
			if err != nil {
				return vr, err
			}
			vr.UploadTmpfile = tf2.Name()
			tf2.Close()
			log.Info().Str("dst", vr.UploadTmpfile).Str("src", readerPath).Msg("fetch: copying extracted nested zip file for upload")
			if err := copyFileContents(vr.UploadTmpfile, readerPath); err != nil {
				return vr, err
			}
		}
		return vr, nil
	}
	result, err := ffetch(atx, feed, opts, cb)
	return fv, result, err
}

func createFeedStats(atx tldb.Adapter, reader *tlcsv.Reader, fvid int) error {
	// Get FeedVersionFileInfos
	fvfis, err := dmfr.NewFeedVersionFileInfosFromReader(reader)
	if err != nil {
		return err
	}
	for _, fvfi := range fvfis {
		fvfi.UpdateTimestamps()
		fvfi.FeedVersionID = fvid
		if _, err := atx.Insert(&fvfi); err != nil {
			return err
		}
	}
	// Get service statistics
	fvsls, err := dmfr.NewFeedVersionServiceInfosFromReader(reader)
	if err != nil {
		return err
	}
	// Batch insert
	bt := make([]interface{}, len(fvsls))
	for i := range fvsls {
		fvsls[i].FeedVersionID = fvid
		bt[i] = &fvsls[i]
	}
	if err := atx.CopyInsert(bt); err != nil {
		return err
	}

	return nil
}
