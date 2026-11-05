package fetch

import (
	"context"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/stretchr/testify/assert"
)

func TestSpoolKey(t *testing.T) {
	at := time.Date(2026, 10, 28, 14, 59, 58, 0, time.FixedZone("PDT", -7*3600))
	tcs := []struct {
		name string
		feed string
		want string
	}{
		{"onestop id", "f-sf~bay~area~rg~rt", "url_type=realtime_vehicle_positions/date=2026-10-28/hour=21/f-sf~bay~area~rg~rt-20261028T215958Z-abc.pb"},
		// A feed id with path characters stays one file name in its hour.
		{"url", "http://host:8080/x", "url_type=realtime_vehicle_positions/date=2026-10-28/hour=21/http%3A%2F%2Fhost%3A8080%2Fx-20261028T215958Z-abc.pb"},
		{"dots", "../up", "url_type=realtime_vehicle_positions/date=2026-10-28/hour=21/..%2Fup-20261028T215958Z-abc.pb"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			key := spoolKey(tc.feed, "realtime_vehicle_positions", "abc", at)
			assert.Equal(t, tc.want, key)
			unescaped, err := url.PathUnescape(escapeKeySegment(tc.feed))
			assert.NoError(t, err)
			assert.Equal(t, tc.feed, unescaped)

			// The key parses back into what made it.
			dir, name := path.Split(key)
			urlType, hour, err := ParseSpoolHourPrefix(strings.TrimSuffix(dir, "/"))
			assert.NoError(t, err)
			assert.Equal(t, "realtime_vehicle_positions", urlType)
			assert.Equal(t, time.Date(2026, 10, 28, 21, 0, 0, 0, time.UTC), hour)
			parsed, err := ParseSpoolName(name)
			assert.NoError(t, err)
			assert.Equal(t, SpoolName{FeedID: tc.feed, FetchedAt: at.UTC(), SHA1: "abc"}, parsed)
		})
	}
}

func TestParseSpoolNameRejects(t *testing.T) {
	for _, name := range []string{"x.pb", "feed-abc.pb", "feed-2026-abc.pb", "feed-20261028T215958Z-abc.tmp", "-20261028T215958Z-abc.pb", ".upload-123"} {
		_, err := ParseSpoolName(name)
		assert.Error(t, err, name)
	}
	_, _, err := ParseSpoolHourPrefix("url_type=x/date=2026-10-28")
	assert.Error(t, err)
}

func TestValidateRTSpool(t *testing.T) {
	assert.NoError(t, ValidateRTSpool("", []string{"anything"}), "no spool, nothing to check")
	assert.NoError(t, ValidateRTSpool(t.TempDir(), []string{"realtime_vehicle_positions", "realtime_trip_updates"}))
	assert.ErrorContains(t, ValidateRTSpool(t.TempDir(), []string{"realtime_vehicle_position"}), "unknown RT spool URL type")
}

// stallStore never finishes an upload until released.
type stallStore struct {
	release chan struct{}
}

func (s stallStore) Upload(context.Context, string, io.Reader) error {
	<-s.release
	return nil
}
func (stallStore) Download(context.Context, string) (io.ReadCloser, int, error) {
	return nil, 0, nil
}
func (stallStore) DownloadAuth(context.Context, string, dmfr.FeedAuthorization) (io.ReadCloser, int, error) {
	return nil, 0, nil
}
func (stallStore) SetSecret(dmfr.Secret) error                        { return nil }
func (stallStore) ListKeys(context.Context, string) ([]string, error) { return nil, nil }

func TestSpoolFileDoesNotWaitOnAStalledStore(t *testing.T) {
	defer func(d time.Duration) { spoolTimeout = d }(spoolTimeout)
	spoolTimeout = 50 * time.Millisecond
	store := stallStore{release: make(chan struct{})}
	defer close(store.release)
	spoolStores.Store("stall://test", store)
	defer spoolStores.Delete("stall://test")

	fn := filepath.Join(t.TempDir(), "msg.pb")
	assert.NoError(t, os.WriteFile(fn, []byte("message"), 0o644))
	start := time.Now()
	spoolFile(context.Background(), "stall://test", "key.pb", fn)
	assert.Less(t, time.Since(start), time.Second, "the fetch moves on once the spool timeout passes")
}
