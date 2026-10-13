package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/copier"
	internaltestdb "github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/server/dbutil"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/jmoiron/sqlx"
)

// Test helpers

var testdb *sqlx.DB

func CheckEnv(key string) (string, string, bool) {
	g := os.Getenv(key)
	if g == "" {
		return "", fmt.Sprintf("%s is not set, skipping", key), false
	}
	return g, "", true
}

func CheckTestDB() (string, bool) {
	_, a, ok := CheckEnv("TL_TEST_SERVER_DATABASE_URL")
	return a, ok
}

func MustOpenTestDB(t testing.TB) *sqlx.DB {
	if testdb != nil {
		return testdb
	}
	dburl := os.Getenv("TL_TEST_SERVER_DATABASE_URL")
	var err error
	testdb, err = dbutil.OpenDB(dburl)
	if err != nil {
		t.Fatal(err)
		return nil
	}
	return testdb
}

// CopyTestFeed copies a feed into a new feed version within tx, deduplicating
// journey patterns as the server's imports do, and returns the feed version's id.
func CopyTestFeed(t testing.TB, tx *sqlx.Tx, reader adapters.Reader) int {
	t.Helper()
	atx := postgres.NewPostgresAdapterFromDBX(tx)
	fv := internaltestdb.CreateTestFeedVersion(atx, "")
	writer := &tldb.Writer{Adapter: atx, FeedVersionID: fv.ID}
	result, err := copier.CopyWithOptions(context.Background(), reader, writer, copier.Options{DeduplicateJourneyPatterns: true})
	if err != nil {
		t.Fatal(err)
	}
	for fn, n := range result.SkipEntityErrorCount {
		t.Fatalf("skipped %d %s entities with errors", n, fn)
	}
	for fn, n := range result.SkipEntityReferenceCount {
		t.Fatalf("skipped %d %s entities with reference errors", n, fn)
	}
	return fv.ID
}
