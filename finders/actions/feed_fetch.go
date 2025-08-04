package actions

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/fetch"
	"github.com/interline-io/transitland-lib/internal/gbfs"
	"github.com/interline-io/transitland-lib/model"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/interline-io/transitland-mw/auth/authn"
	"github.com/interline-io/transitland-mw/auth/authz"
	"google.golang.org/protobuf/proto"
)

func StaticFetch(ctx context.Context, feedId string, feedSrc io.Reader, feedUrl string) (*model.FeedVersionFetchResult, error) {
	cfg := model.ForContext(ctx)
	dbf := cfg.Finder

	urlType := "static_current"
	feed, err := fetchCheckFeed(ctx, feedId)
	if err != nil {
		return nil, err
	}
	if feed == nil {
		return nil, nil
	}
	if feedUrl == "" {
		feedUrl = feed.URLs.StaticCurrent
	}

	// Prepare
	fetchOpts := fetch.StaticFetchOptions{
		Options: fetch.Options{
			FeedID:        feed.ID,
			URLType:       urlType,
			FeedURL:       feedUrl,
			Storage:       cfg.Storage,
			Secrets:       cfg.Secrets,
			FetchedAt:     time.Now().In(time.UTC),
			AllowFTPFetch: true,
		},
	}
	if user := authn.ForContext(ctx); user != nil {
		fetchOpts.CreatedBy.Set(user.ID())
	}

	// Allow a Reader
	if feedSrc != nil {
		tmpfile, err := os.CreateTemp("", "validator-upload")
		if err != nil {
			return nil, err
		}
		io.Copy(tmpfile, feedSrc)
		tmpfile.Close()
		defer os.Remove(tmpfile.Name())
		fetchOpts.FeedURL = tmpfile.Name()
		fetchOpts.AllowLocalFetch = true
	}

	// Make request
	mr := model.FeedVersionFetchResult{}
	db := postgres.NewPostgresAdapterFromDBX(dbf.DBX())
	if err := db.Tx(func(atx tldb.Adapter) error {
		fr, err := fetch.StaticFetch(ctx, atx, fetchOpts)
		if err != nil {
			return err
		}
		mr.FoundSha1 = fr.Found
		if fr.FetchError != nil {
			a := fr.FetchError.Error()
			mr.FetchError = &a
		} else if fr.FeedVersion != nil {
			mr.FeedVersion = &model.FeedVersion{FeedVersion: *fr.FeedVersion}
			mr.FetchError = nil
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &mr, nil
}

func RTFetch(ctx context.Context, target string, feedId string, feedUrl string, urlType string) error {
	cfg := model.ForContext(ctx)

	feed, err := fetchCheckFeed(ctx, feedId)
	if err != nil {
		return err
	}
	if feed == nil {
		return nil
	}

	// Prepare
	fetchOpts := fetch.RTFetchOptions{
		Options: fetch.Options{
			FeedID:    feed.ID,
			URLType:   urlType,
			FeedURL:   feedUrl,
			Storage:   cfg.RTStorage,
			Secrets:   cfg.Secrets,
			FetchedAt: time.Now().In(time.UTC),
		},
	}

	// Make request
	var rtMsg *pb.FeedMessage
	var fetchErr error
	if err := postgres.NewPostgresAdapterFromDBX(cfg.Finder.DBX()).Tx(func(atx tldb.Adapter) error {
		fr, err := fetch.RTFetch(ctx, atx, fetchOpts)
		if err != nil {
			return err
		}
		rtMsg = fr.Message
		fetchErr = fr.FetchError
		return nil
	}); err != nil {
		return err
	}

	// Check result and cache
	if fetchErr != nil {
		return fetchErr
	}
	rtdata, err := proto.Marshal(rtMsg)
	if err != nil {
		return errors.New("invalid rt data")
	}
	key := fmt.Sprintf("rtdata:%s:%s", target, urlType)
	return cfg.RTFinder.AddData(ctx, key, rtdata)
}

func GbfsFetch(ctx context.Context, feedId string, feedUrl string) error {
	cfg := model.ForContext(ctx)
	gfeeds, err := cfg.Finder.FindFeeds(ctx, nil, nil, nil, &model.FeedFilter{OnestopID: &feedId})
	if err != nil {
		log.For(ctx).Error().Err(err).Msg("gbfs-fetch: error loading source feed")
		return err
	}
	if len(gfeeds) == 0 {
		log.For(ctx).Error().Err(err).Msg("gbfs-fetch: source feed not found")
		return errors.New("feed not found")
	}

	// Make request
	opts := gbfs.Options{}
	opts.FeedURL = gfeeds[0].URLs.GbfsAutoDiscovery
	opts.FeedID = gfeeds[0].ID
	opts.URLType = "gbfs_auto_discovery"
	opts.FetchedAt = time.Now().In(time.UTC)
	if feedUrl != "" {
		opts.FeedURL = feedUrl
	}
	feeds, result, err := gbfs.Fetch(
		ctx,
		postgres.NewPostgresAdapterFromDBX(cfg.Finder.DBX()),
		opts,
	)
	if err != nil {
		return err
	}
	if result.FetchError != nil {
		return result.FetchError
	}

	// Save to cache
	for _, feed := range feeds {
		if feed.SystemInformation != nil {
			key := fmt.Sprintf("%s:%s", feedId, feed.SystemInformation.Language.Val)
			cfg.GbfsFinder.AddData(ctx, key, feed)
		}
	}
	return nil
}

func fetchCheckFeed(ctx context.Context, feedId string) (*model.Feed, error) {
	// Check feed exists
	cfg := model.ForContext(ctx)
	feeds, err := cfg.Finder.FindFeeds(ctx, nil, nil, nil, &model.FeedFilter{OnestopID: &feedId})
	if err != nil {
		return nil, err
	}
	if len(feeds) == 0 {
		return nil, errors.New("feed not found")
	}
	feed := feeds[0]

	// Check feed permissions
	if checker := cfg.Checker; checker == nil {
		// pass
	} else if check, err := checker.FeedPermissions(ctx, &authz.FeedRequest{Id: int64(feed.ID)}); err != nil {
		return nil, err
	} else if !check.Actions.CanCreateFeedVersion {
		return nil, errors.New("unauthorized")
	}
	return feed, nil
}
