package testutil

import (
	"fmt"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/server/dbutil"
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
