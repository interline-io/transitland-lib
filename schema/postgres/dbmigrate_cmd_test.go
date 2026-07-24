package postgres

import (
	"testing"

	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPendingVersions(t *testing.T) {
	available := []int{100, 200, 300}
	tcs := []struct {
		name    string
		current int
		want    []int
	}{
		{"fresh database", -1, []int{100, 200, 300}},
		{"partially applied", 100, []int{200, 300}},
		{"one behind", 200, []int{300}},
		{"up to date", 300, nil},
		{"ahead of binary", 400, nil},
		{"between versions", 250, []int{300}},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, pendingVersions(tc.current, available))
		})
	}
	assert.Nil(t, pendingVersions(-1, nil), "no embedded migrations means never behind")
}

func TestAvailableVersions(t *testing.T) {
	src, err := iofs.New(EmbeddedMigrations, "migrations")
	require.NoError(t, err)
	got, err := availableVersions(src)
	require.NoError(t, err)
	require.NotEmpty(t, got, "expected embedded migrations")
	for i := 1; i < len(got); i++ {
		assert.Greater(t, got[i], got[i-1], "versions must be strictly ascending and unique")
	}
}
