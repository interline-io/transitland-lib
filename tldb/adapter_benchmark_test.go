package tldb

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/testutil"
)

// Tests adapter Insert performance.
func Benchmark_Adapter_Insert(b *testing.B) {
	ctx := context.TODO()
	for k, v := range testAdapters {
		b.Run(k, func(b *testing.B) {
			adapter := v()
			if err := adapter.Open(); err != nil {
				b.Error(err)
			}
			if err := adapter.Create(); err != nil {
				b.Error(err)
			}
			feedid, err := adapter.Insert(ctx, &dmfr.Feed{FeedID: fmt.Sprintf("%d", time.Now().UnixNano())})
			if err != nil {
				b.Error(err)
			}
			b.ResetTimer()
			ent := dmfr.FeedVersion{FeedID: feedid}
			for i := 0; i < b.N; i++ {
				_, err := adapter.Insert(ctx, &ent)
				if err != nil {
					b.Error(err)
				}
			}
		})
	}
}

// Tests raw database performance.
func Benchmark_Adapter_InsertRaw(b *testing.B) {
	ctx := context.TODO()
	for k, v := range testAdapters {
		b.Run(k, func(b *testing.B) {
			adapter := v()
			if err := adapter.Open(); err != nil {
				b.Error(err)
			}
			if err := adapter.Create(); err != nil {
				b.Error(err)
			}
			feedid, err := adapter.Insert(ctx, &dmfr.Feed{FeedID: fmt.Sprintf("%d", time.Now().UnixNano())})
			if err != nil {
				b.Error(err)
			}
			b.ResetTimer()
			ent := dmfr.FeedVersion{FeedID: feedid}
			q := adapter.DBX().Rebind(`INSERT INTO feed_versions(feed_id, file, earliest_calendar_date, latest_calendar_date, sha1, sha1_dir,fetched_at, created_at, updated_at, url) VALUES (?,?,?,?,?,?,?,?,?,?,?)`)
			for i := 0; i < b.N; i++ {
				_, err := adapter.DBX().ExecContext(
					ctx,
					q,
					ent.FeedID,
					ent.File,
					ent.EarliestCalendarDate,
					ent.LatestCalendarDate,
					ent.SHA1,
					ent.SHA1Dir,
					ent.FetchedAt,
					ent.CreatedAt,
					ent.UpdatedAt,
					ent.URL,
				)
				if err != nil {
					b.Error(err)
				}
			}
		})
	}
}

// Tests multiple insert performance
// There is a lot of setup in this test because we need a FeedVersion, Trip, and Stop
func Benchmark_Adapter_MultiInsert(b *testing.B) {
	ctx := context.TODO()
	for k, v := range testAdapters {
		b.Run(k, func(b *testing.B) {
			adapter := v()
			if err := adapter.Open(); err != nil {
				b.Error(err)
			}
			if err := adapter.Create(); err != nil {
				b.Error(err)
			}
			// Load the minimal test feed...
			writer := Writer{Adapter: adapter}
			_, reader := testutil.NewMinimalTestFeed()
			if err := reader.Open(); err != nil {
				b.Error(err)
			}
			if err := testutil.DirectCopy(reader, &writer); err != nil {
				b.Error(err)
			}
			// get ids
			fvid := 0
			tripid := 0
			stopid := 0
			if err := adapter.DBX().QueryRowxContext(ctx, "SELECT id FROM feed_versions LIMIT 1").Scan(&fvid); err != nil {
				b.Error(err)
			}
			if err := adapter.DBX().QueryRowxContext(ctx, "SELECT id FROM gtfs_trips LIMIT 1").Scan(&tripid); err != nil {
				b.Error(err)
			}
			if err := adapter.DBX().QueryRowxContext(ctx, "SELECT id FROM gtfs_stops LIMIT 1").Scan(&stopid); err != nil {
				b.Error(err)
			}
			if _, err := adapter.DBX().ExecContext(ctx, adapter.DBX().Rebind("DELETE FROM gtfs_stop_times WHERE trip_id = ?"), tripid); err != nil {
				b.Error(err)
			}
			// Reset the timer
			b.ResetTimer()
			count := 1000
			for i := 0; i < b.N; i++ {
				// Make the StopTimes
				ents := make([]interface{}, 0)
				for i := 0; i < 1000; i++ {
					count++
					ent := gtfs.StopTime{}
					ent.StopSequence.Set(int64(count))
					ent.StopID.Set(strconv.Itoa(stopid))
					ent.TripID.Set(strconv.Itoa(tripid))
					ent.FeedVersionID = fvid
					ents = append(ents, &ent)
				}
				if _, err := adapter.MultiInsert(ctx, ents); err != nil {
					b.Error(err)
				}
			}
			if err := adapter.Close(); err != nil {
				b.Error(err)
			}
			if err := reader.Close(); err != nil {
				b.Error(err)
			}
		})
	}
}
