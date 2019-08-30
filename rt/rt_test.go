package rt

import (
	"encoding/json"
	"fmt"
	"testing"
)

func Test_msgstats(t *testing.T) {
	msg, err := readmsg("../testdata/rt/example.pb")
	if err != nil {
		t.Error(err)
	}
	msgstats(msg)
}

func Test_readmsg(t *testing.T) {
	msg, err := readmsg("../testdata/rt/example.pb")
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
	z, _ := json.Marshal(msg)
	fmt.Printf("%s\n", z)
}

func Test_readmsg_error(t *testing.T) {
	_, err := readmsg("../testdata/example.zip")
	if err == nil {
		t.Errorf("got no error, expected illegal tag")
	}
}
