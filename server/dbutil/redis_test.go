package dbutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_getRedisOpts(t *testing.T) {
	tcs := []struct {
		url  string
		addr string
		db   int
	}{
		{"redis://localhost:6379", "localhost:6379", 0},
		{"redis://localhost:6379/1", "localhost:6379", 1},
		{"redis://localhost", "localhost:6379", 0},
		{"redis://localhost/1", "localhost:6379", 1},
	}
	for _, tc := range tcs {
		opts, err := getRedisOpts(tc.url)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, tc.addr, opts.Addr)
		assert.Equal(t, tc.db, opts.DB)
	}

}
