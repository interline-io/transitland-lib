package rt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/rt/pb"
	"google.golang.org/protobuf/encoding/protojson"
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

func RTAsJson(msg *pb.FeedMessage) ([]byte, error) {
	m := protojson.MarshalOptions{Indent: "  "}
	jdata, err := m.Marshal(msg)
	return jdata, err
}

func TestRTAsJson(t *testing.T) {
	msg, err := ReadFile(testutil.RelPath("test/data/rt/ct-vehicle-positions.pb"))
	if err != nil {
		t.Fatal(err)
	}
	jdata, err := RTAsJson(msg)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(jdata))
}

// func Test_convert(t *testing.T) {
// 	testConvert(t, testutil.RelPath("test/data/rt"))
// }

func testConvert(t *testing.T, dir string) {
	fns, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, fs := range fns {
		fn := fs.Name()
		newFn := filepath.Join(dir, fmt.Sprintf("%s.json", fn))
		if !strings.HasSuffix(fn, ".pb") {
			continue
		}
		msg, err := ReadFile(filepath.Join(dir, fn))
		if err != nil {
			t.Log(fn, err.Error())
			continue
		}
		jdata, err := RTAsJson(msg)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(newFn, jdata, 0644); err != nil {
			t.Fatal(err)
		}
	}
}
