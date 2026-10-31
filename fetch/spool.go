package fetch

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/request"
)

// RTURLTypes are the URL types an RT fetch can be for, and so the ones that can
// be spooled.
var RTURLTypes = []string{"realtime_alerts", "realtime_trip_updates", "realtime_vehicle_positions"}

// spoolTimeout bounds how long a fetch waits on the spool. A slow or stalled
// spool must not hold up the fetch, whose message still has to reach the
// realtime cache; a write still running when it passes finishes in the
// background.
var spoolTimeout = 5 * time.Second

// ValidateRTSpool checks spool settings at startup, so a mistyped URL type or
// store fails then rather than spooling nothing, or failing on every fetch.
func ValidateRTSpool(spool string, urlTypes []string) error {
	if spool == "" {
		return nil
	}
	for _, t := range urlTypes {
		if !slices.Contains(RTURLTypes, t) {
			return fmt.Errorf("unknown RT spool URL type %q, expected one of: %s", t, strings.Join(RTURLTypes, ", "))
		}
	}
	_, err := spoolStore(spool)
	return err
}

// spoolKey builds a spool object key. Partitioned by URL type, date and hour
// rather than by feed, so a batch job reads one hour of every feed from one
// place. The feed, time and sha1 make the file name, in that order; the time
// has no dashes, so the name splits from the right.
//
// The hour is the fetch's start, so a fetch running across the hour lands in
// the hour it began, after that hour has ended: a reader should allow a grace
// period, or reread recent hours.
func spoolKey(onestopID, urlType, sha1 string, fetchedAt time.Time) string {
	t := fetchedAt.UTC()
	return fmt.Sprintf("url_type=%s/date=%s/hour=%s/%s-%s-%s.pb",
		urlType, t.Format("2006-01-02"), t.Format("15"), escapeKeySegment(onestopID), t.Format("20060102T150405Z"), sha1)
}

// escapeKeySegment percent-encodes every byte outside [A-Za-z0-9~._-], so a
// feed id with a slash, colon or other path character stays one file name in
// its hour. url.PathUnescape reverses it.
func escapeKeySegment(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case 'a' <= c && c <= 'z', 'A' <= c && c <= 'Z', '0' <= c && c <= '9', c == '~', c == '.', c == '_', c == '-':
			b.WriteByte(c)
		default:
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}

// spoolStores holds one store per spool URL, rather than one per fetch.
var spoolStores sync.Map

func spoolStore(spool string) (request.Store, error) {
	if s, ok := spoolStores.Load(spool); ok {
		return s.(request.Store), nil
	}
	s, err := request.GetStore(spool)
	if err != nil {
		return nil, err
	}
	actual, _ := spoolStores.LoadOrStore(spool, s)
	return actual.(request.Store), nil
}

// spoolFile copies a fetched file to the spool, waiting at most spoolTimeout.
// Failures are logged, not returned: the spool is a copy for batch jobs, and
// the fetch carries on without it.
func spoolFile(ctx context.Context, spool string, key string, fn string) {
	// Read now: the file is removed when the fetch returns, and the write may
	// outlive it.
	data, err := os.ReadFile(fn)
	if err != nil {
		log.For(ctx).Error().Err(err).Str("key", key).Msg("rt fetch: spool read failed")
		return
	}
	done := make(chan error, 1)
	go func() {
		store, err := spoolStore(spool)
		if err == nil {
			err = store.Upload(context.WithoutCancel(ctx), key, bytes.NewReader(data))
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			log.For(ctx).Error().Err(err).Str("key", key).Msg("rt fetch: spool write failed")
		}
	case <-time.After(spoolTimeout):
		log.For(ctx).Warn().Str("key", key).Dur("timeout", spoolTimeout).Msg("rt fetch: spool write still running; continuing without waiting")
	}
}
