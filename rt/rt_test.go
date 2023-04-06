package rt

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
)

func Test_readmsg(t *testing.T) {
	msg, err := ReadFile(testutil.RelPath("test/data/rt/example.pb"))
	if err != nil {
		t.Error(err)
	}
	t.Run("Timestamp", func(t *testing.T) {
		exp := uint64(1559008978)
		if got := msg.GetHeader().GetTimestamp(); got != exp {
			t.Errorf("got %d expect %d", got, exp)
		}
	})
	t.Run("EntityCount", func(t *testing.T) {
		exp := 26
		if got := len(msg.Entity); got != exp {
			t.Errorf("got %d entities, expect %d", got, exp)
		}
	})
	t.Run("Entity", func(t *testing.T) {
		pbents := msg.Entity
		if len(pbents) == 0 {
			t.Error("no message entities")
			return
		}
		ent := pbents[0]
		exp := "2211905WKDY"
		if got := ent.GetId(); got != exp {
			t.Errorf("got '%s' expect '%s'", got, exp)
		}
	})
	// z, _ := json.Marshal(msg)
	// t.Logf("%s\n", z)
}

func TestReadFileError(t *testing.T) {
	_, err := ReadFile(testutil.RelPath("test/data/example.zip"))
	if err == nil {
		t.Errorf("got no error, expected illegal tag")
	}
}
