package tlcsv

import (
	"os"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/interline-io/log"
	"github.com/rs/zerolog"
)

// TestTripsWithStopTimesMem drains TripsWithStopTimes with nothing attached —
// no copier, writer, or validators — to measure the reader join's baseline
// memory in isolation. It skips unless TL_BENCH_FEED points at a feed (a
// directory or .zip). Run it under gctrace and graph the output exactly like
// the copy/validate logs:
//
//	TL_BENCH_FEED=feed.zip TL_GTFS_CHUNKSIZE=250000 TL_LOG=trace GODEBUG=gctrace=1 \
//	  go test -run TestTripsWithStopTimesMem -count=1 -timeout 30m ./tlcsv/ 2>&1 | tee reader.log
//
// The reader's grouped fast path should hold only one trip's stop_times plus
// one chunk's trips, so the curve should be flat regardless of feed size.
func TestTripsWithStopTimesMem(t *testing.T) {
	path := os.Getenv("TL_BENCH_FEED")
	if path == "" {
		t.Skip("set TL_BENCH_FEED to a feed (dir or .zip) to run the reader memory benchmark")
	}
	log.SetLevel(zerolog.TraceLevel) // emit "tlcsv: read pass" + "processing trip chunk" lines

	reader, err := NewReader(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Open(); err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	// Track peak HeapAlloc while draining.
	var peak uint64
	done := make(chan struct{})
	go func() {
		var ms runtime.MemStats
		tk := time.NewTicker(200 * time.Millisecond)
		defer tk.Stop()
		for {
			select {
			case <-done:
				return
			case <-tk.C:
				runtime.ReadMemStats(&ms)
				if v := atomic.LoadUint64(&peak); ms.HeapAlloc > v {
					atomic.StoreUint64(&peak, ms.HeapAlloc)
				}
			}
		}
	}()

	var trips, stopTimes int64
	for tst := range reader.TripsWithStopTimes() {
		trips++
		stopTimes += int64(len(tst.StopTimes))
	}
	close(done)

	t.Logf("reader-only drain: trips=%d stop_times=%d peak_heap_alloc=%dMB",
		trips, stopTimes, atomic.LoadUint64(&peak)/(1024*1024))
}
