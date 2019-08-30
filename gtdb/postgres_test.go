package gtdb

import (
	"testing"
)

func TestPostgresAdapter(t *testing.T) {
	if adapter, ok := getTestAdapters()["PostgresAdapter"]; ok {
		testAdapter(t, adapter())
	} else {
		t.Skip("no test url for PostgresAdapter")
	}
}
