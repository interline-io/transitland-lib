package jobserver

import (
	"log"
	"os"
	"testing"

	"github.com/interline-io/transitland-mw/testutil"
)

func TestMain(m *testing.M) {
	if a, ok := testutil.CheckTestDB(); !ok {
		log.Print(a)
		return
	}
	os.Exit(m.Run())
}
