package tlpb

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
)

func TestReadStops(t *testing.T) {
	ReadStops(testutil.RelPath("test/data/external/bart.zip"))
}
