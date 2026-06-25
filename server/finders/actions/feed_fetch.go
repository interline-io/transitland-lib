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
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/authz"
	"github.com/interline-io/transitland-lib/server/model"
	"google.golang.org/protobuf/proto"
)

func StaticFetch(ctx context.Context, feedId string, feedSrc io.Reader, feedUrl string) (*model.FeedVersionFetchResult, error) {
	cfg := model.ForContext(ctx)

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
			FeedID:                   feed.ID,
			URLType:                  urlType,
			FeedURL:                  feedUrl,
			Storage:                  cfg.Storage,
			Secrets:                  cfg.Secrets,
			FetchedAt:                time.Now().In(time.UTC),
			AllowFTPFetch:            true,
			AllowHTTPFetchUnfiltered: cfg.AllowHTTPFetchUnfiltered,
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
	fr, err := fetch.StaticFetch(ctx, cfg.FeedManager, fetchOpts)
	if err != nil {
		return nil, err
	}
	mr.FoundSha1 = fr.Found
	if fr.FetchError != nil {
		a := fr.FetchError.Error()
		mr.FetchError = &a
	} else if fr.FeedVersion != nil {
		mr.FeedVersion = &model.FeedVersion{FeedVersion: *fr.FeedVersion}
		mr.FetchError = nil
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

	// Archive to RTStorage only when this feed opts in (rt_retention_period > 0).
	archiveStorage := ""
	if cfg.RTStorage != "" {
		if states, errs := cfg.Finder.FeedStatesByFeedIDs(ctx, []int{feed.ID}); len(errs) > 0 && errs[0] != nil {
			log.For(ctx).Error().Err(errs[0]).Int("feed_id", feed.ID).Msg("rt-fetch: could not load feed state for archive policy; not archiving")
		} else if len(states) > 0 && states[0] != nil && states[0].RTRetentionPeriod > 0 {
			archiveStorage = cfg.RTStorage
		}
	}

	// Prepare
	fetchOpts := fetch.RTFetchOptions{
		Options: fetch.Options{
			FeedID:                   feed.ID,
			URLType:                  urlType,
			FeedURL:                  feedUrl,
			Storage:                  archiveStorage,
			Secrets:                  cfg.Secrets,
			FetchedAt:                time.Now().In(time.UTC),
			AllowHTTPFetchUnfiltered: cfg.AllowHTTPFetchUnfiltered,
		},
	}

	// Make request
	fr, err := fetch.RTFetch(ctx, cfg.FeedManager, fetchOpts)
	if err != nil {
		return err
	}
	rtMsg := fr.Message
	fetchErr := fr.FetchError

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
	opts.AllowHTTPFetchUnfiltered = cfg.AllowHTTPFetchUnfiltered
	if feedUrl != "" {
		opts.FeedURL = feedUrl
	}
	feeds, result, err := gbfs.Fetch(
		ctx,
		cfg.Adapter,
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
	cfg := model.ForContext(ctx)

	if cfg.Checker == nil {
		log.For(ctx).Debug().Str("feed_id", feedId).Msg("fetchCheckFeed: no Checker configured")
		return nil, authz.ErrUnauthorized
	}

	// Both not-found and not-authorized return ErrUnauthorized — distinguishing
	// them would let unauthorized callers probe feed existence.
	feeds, err := cfg.Finder.FindFeeds(ctx, nil, nil, nil, &model.FeedFilter{OnestopID: &feedId})
	if err != nil {
		return nil, err
	}
	if len(feeds) == 0 {
		log.For(ctx).Debug().Str("feed_id", feedId).Msg("fetchCheckFeed: feed not found")
		return nil, authz.ErrUnauthorized
	}
	feed := feeds[0]

	ok, err := cfg.Checker.Check(ctx, authz.ObjectRef{Type: authz.FeedType, ID: int64(feed.ID)}, authz.CanCreateFeedVersion)
	if err != nil {
		return nil, err
	}
	if !ok {
		log.For(ctx).Debug().Str("feed_id", feedId).Int("feed_db_id", feed.ID).Msg("fetchCheckFeed: caller not authorized")
		return nil, authz.ErrUnauthorized
	}
	return feed, nil
}
