package copier_test

import (
	"context"
	"os"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/ext/builders"
	"github.com/interline-io/transitland-lib/tlcsv"
)

// runCopierBench drains a feed through a Copier with the given options, writing to a
// null writer, and logs the entity counts and peak HeapAlloc. Skips unless
// TL_BENCH_FEED points at a feed (dir or .zip). Run under gctrace and graph the output
// like the copy/validate logs:
//
//	TL_BENCH_FEED=feed.zip TL_GTFS_CHUNKSIZE=250000 TL_LOG=trace GODEBUG=gctrace=1 \
//	  go test -run TestCopier...Mem -count=1 -timeout 1800s ./copier/ 2>&1 | tee out.log
func runCopierBench(t *testing.T, opts copier.Options) {
	path := os.Getenv("TL_BENCH_FEED")
	if path == "" {
		t.Skip("set TL_BENCH_FEED to a feed (dir or .zip) to run the copier memory benchmark")
	}
	ctx := context.Background()

	reader, err := tlcsv.NewReader(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Open(); err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	cp, err := copier.NewCopier(ctx, reader, &empty.Writer{}, opts)
	if err != nil {
		t.Fatal(err)
	}

	// Track peak HeapAlloc while copying.
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

	result, err := cp.Copy(ctx)
	close(done)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s: trips=%d stop_times=%d peak_heap_alloc=%dMB", t.Name(),
		result.EntityCount["trips.txt"], result.EntityCount["stop_times.txt"],
		atomic.LoadUint64(&peak)/(1024*1024))
}

// TestCopierBaselineMem is the floor: no validators, extensions, or filters, shape
// caching off — the copier's own EntityMap + pattern tables above the reader.
func TestCopierBaselineMem(t *testing.T) {
	runCopierBench(t, copier.Options{NoValidators: true, NoShapeCache: true})
}

// TestCopierImportDefaultsMem is the ceiling: the full default stack an import runs —
// the minimal validators, the shape geometry cache, and DefaultImportBuilders (route
// geometries, route stops, headways, convex hulls, agency places) — against a null
// writer, so it exercises everything a real import does without a database.
func TestCopierImportDefaultsMem(t *testing.T) {
	opts := copier.Options{}
	for _, b := range builders.DefaultImportBuilders() {
		opts.AddExtension(b)
	}
	runCopierBench(t, opts)
}
