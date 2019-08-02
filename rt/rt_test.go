package rt

import (
	"testing"
)

func Test_readmsg(t *testing.T) {
	msg, err := readmsg("../testdata/rt/example.pbf")
	if err != nil {
		t.Error(err)
	}
	t.Run("Timestamp", func(t *testing.T) {
		exp := uint64(1559008978)
		if msg.Header.Timestamp != nil && uint64(*msg.Header.Timestamp) != exp {
			got := uint64(*msg.Header.Timestamp)
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
		if ent.Id != nil && *ent.Id != exp {
			t.Errorf("got '%s' expect '%s'", *ent.Id, exp)
		}
	})
	// z, _ := json.Marshal(msg)
	// fmt.Printf("%s\n", z)
}

func Test_readmsg_error(t *testing.T) {
	_, err := readmsg("../testdata/example.zip")
	if err == nil {
		t.Errorf("got no error, expected illegal tag")
	}
}
