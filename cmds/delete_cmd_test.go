package cmds

import (
	"context"
	"strconv"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/stretchr/testify/assert"
)

func TestDeleteCommand_Run(t *testing.T) {
	ctx := context.Background()

	t.Run("errors when no requested feed versions exist", func(t *testing.T) {
		atx := testdb.TempSqliteAdapter()
		cmd := DeleteCommand{
			FVArgs:  FeedVersionArgs{FVIDs: []string{"999999"}},
			Adapter: atx,
		}
		err := cmd.Run(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no matching feed versions")
	})

	t.Run("deletes an existing feed version", func(t *testing.T) {
		atx := testdb.TempSqliteAdapter()
		fv := testdb.CreateTestFeedVersion(atx, "delete-test.zip")
		cmd := DeleteCommand{
			FVArgs:  FeedVersionArgs{FVIDs: []string{strconv.Itoa(fv.ID)}},
			Adapter: atx,
		}
		assert.NoError(t, cmd.Run(ctx))
	})
}
