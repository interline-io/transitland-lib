package fetch

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/download"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
)

// Options sets options for a fetch operation.
type Options struct {
	FeedURL                 string
	FeedID                  string
	FeedCreate              bool
	IgnoreDuplicateContents bool
	Directory               string
	S3                      string
	FetchedAt               time.Time
	Secrets                 download.Secrets
	CreatedBy               tl.OString
	Name                    tl.OString
	Description             tl.OString
}

// Result contains results of a fetch operation.
type Result struct {
	FeedVersion  tl.FeedVersion
	Path         string
	FoundSHA1    bool
	FoundDirSHA1 bool
	FetchError   error
}

// DatabaseFetch fetches and creates a new FeedVersion for a given Feed.
// An error return from this function is a serious failure.
// Saves FeedState.LastFetchError for regular failures.
func DatabaseFetch(atx tldb.Adapter, opts Options) (Result, error) {
	fr := Result{}
	// Get feed, create if not present and FeedCreate is specified
	tlfeed := tl.Feed{}
	if err := atx.Get(&tlfeed, `SELECT * FROM current_feeds WHERE onestop_id = ?`, opts.FeedID); err == sql.ErrNoRows && opts.FeedCreate {
		tlfeed.FeedID = opts.FeedID
		tlfeed.Spec = "gtfs"
		if tlfeed.ID, err = atx.Insert(&tlfeed); err != nil {
			return fr, err
		}
	} else if err != nil {
		return fr, errors.New("feed does not exist")
	}
	if opts.FeedURL == "" {
		opts.FeedURL = tlfeed.URLs.StaticCurrent
	}
	if opts.FetchedAt.IsZero() {
		opts.FetchedAt = time.Now().UTC()
	}
	// Get state, create if necessary
	tlstate := dmfr.FeedState{FeedID: tlfeed.ID}
	if err := atx.Get(&tlstate, `SELECT * FROM feed_states WHERE feed_id = ?`, tlfeed.ID); err == sql.ErrNoRows {
		tlstate.ID, err = atx.Insert(&tlstate)
		if err != nil {
			return fr, err
		}
	} else if err != nil {
		return fr, err
	}
	// Immediately save LastFetchedAt
	tlstate.LastFetchedAt = tl.OTime{Time: opts.FetchedAt, Valid: true}
	tlstate.LastFetchError = ""
	tlstate.UpdateTimestamps()
	if err := atx.Update(&tlstate, "last_fetched_at", "last_fetch_error"); err != nil {
		return fr, err
	}
	// Start fetching
	fr, err := fetchAndCreateFeedVersion(atx, tlfeed, opts)
	if err != nil {
		return fr, err
	}
	if fr.FetchError != nil {
		tlstate.LastFetchError = fr.FetchError.Error()
	} else {
		tlstate.LastSuccessfulFetchAt = tl.OTime{Time: opts.FetchedAt, Valid: true}
	}
	// else if fr.FoundSHA1 || fr.FoundDirSHA1 {}
	// Save updated timestamps
	tlstate.UpdateTimestamps()
	if err := atx.Update(&tlstate, "last_fetched_at", "last_fetch_error", "last_successful_fetch_at"); err != nil {
		return fr, err
	}
	return fr, nil
}

type canSetAuth interface {
	SetAuth(tl.FeedAuthorization, download.Secret)
}

// fetchAndCreateFeedVersion from a URL.
// Returns an error if a serious failure occurs, such as database or filesystem access.
// Sets Result.FetchError if a regular failure occurs, such as a 404.
// feed is an argument to provide the ID, File, and Authorization.
func fetchAndCreateFeedVersion(atx tldb.Adapter, feed tl.Feed, opts Options) (Result, error) {
	fr := Result{}
	if opts.FeedURL == "" {
		fr.FetchError = errors.New("no url")
		return fr, nil
	}
	// Get secret
	secret := download.Secret{}
	if a, err := opts.Secrets.MatchFeed(opts.FeedID); err == nil {
		secret = a
	} else if a, err := opts.Secrets.MatchFilename(feed.File); err == nil {
		secret = a
	} else if feed.Authorization.Type != "" {
		fr.FetchError = errors.New("no secret found")
		return fr, nil
	}
	// Get reader
	reader, err := tlcsv.NewReader(opts.FeedURL)
	if err != nil {
		fr.FetchError = err
		return fr, nil
	}
	if v, ok := reader.Adapter.(canSetAuth); ok {
		v.SetAuth(feed.Authorization, secret)
	}
	// Open
	if err := reader.Open(); err != nil {
		fr.FetchError = err
		return fr, nil
	}
	defer reader.Close()
	// Get initialized FeedVersion
	fv, err := tl.NewFeedVersionFromReader(reader)
	if err != nil {
		fr.FetchError = err
		return fr, nil
	}
	fv.URL = opts.FeedURL
	fv.FeedID = feed.ID
	fv.FetchedAt = opts.FetchedAt
	fv.CreatedBy = opts.CreatedBy
	fv.Name = opts.Name
	fv.Description = opts.Description
	// Is this SHA1 already present?
	checkfvid := tl.FeedVersion{}
	err = atx.Get(&checkfvid, "SELECT * FROM feed_versions WHERE sha1 = ? OR sha1_dir = ?", fv.SHA1, fv.SHA1Dir)
	if err == nil {
		// Already present
		fr.FeedVersion = checkfvid
		fr.FoundSHA1 = (checkfvid.SHA1 == fv.SHA1)
		fr.FoundDirSHA1 = (checkfvid.SHA1Dir == fv.SHA1Dir)
		return fr, nil
	} else if err == sql.ErrNoRows {
		// Not present, create below
	} else if err != nil {
		// Serious error
		return fr, err
	}
	// Upload file or copy to output directory
	if opts.S3 != "" {
		awscmd := exec.Command(
			"aws",
			"s3",
			"cp",
			reader.Path(),
			fmt.Sprintf("%s/%s.zip", opts.S3, fv.SHA1),
		)
		if output, err := awscmd.Output(); err != nil {
			return fr, fmt.Errorf("upload error: %s: %s", err, output)
		}
	}
	if opts.Directory != "" {
		fn := fv.SHA1 + ".zip"
		outfn := filepath.Join(opts.Directory, fn)
		if err := copyFileContents(reader.Path(), outfn); err != nil {
			return fr, err
		}
		fv.File = fn
		fr.Path = fv.File // TODO: remove
	}
	// Return fv
	fv.UpdateTimestamps()
	fv.ID, err = atx.Insert(&fv)
	fr.FeedVersion = fv
	if err != nil {
		return fr, err
	}
	// Update stats records
	if err := createFeedStats(atx, reader, fv.ID); err != nil {
		return fr, err
	}
	return fr, nil
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

func copyFileContents(src, dst string) (err error) {
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
