package tlpb

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/interline-io/transitland-lib/internal/tlpb/gtfs"
	"github.com/interline-io/transitland-lib/internal/tlpb/pb"
	"github.com/interline-io/transitland-lib/tlcsv"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func printFirst(v []any) {
	if len(v) == 0 {
		return
	}
	fmt.Println(toJson(v[0]))
}
func printAll(v []any) {
	for _, ent := range v {
		fmt.Println(toJson(ent))
	}
}

func pbJson(v protoreflect.ProtoMessage) string {
	jj, _ := protojson.Marshal(v)
	return string(jj)
}

func toJson(v any) string {
	jj, _ := json.Marshal(v)
	return string(jj)
}

var TESTFILE = ""
var TESTTABLE = ""

func init() {
	TESTFILE = testpath.RelPath("test/data/external/bart.zip")
	TESTTABLE = "stops.txt"
}

//////////////////

func TestReadPB(t *testing.T) {
	// ents, err := ReadPB(TESTFILE)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// for _, ent := range ents {
	// 	fmt.Println(ent)
	// }
}

// func BenchmarkReadPB(b *testing.B) {
// 	for n := 0; n < b.N; n++ {
// 		ReadPB(TESTFILE)
// 	}
// }

func ReadPB(fn string) ([]any, error) {
	a := tlcsv.NewZipAdapter(fn)
	if err := a.Open(); err != nil {
		panic(err)
	}
	var ret []any
	err := a.ReadRows(TESTTABLE, func(row tlcsv.Row) {
		ent := &pb.Stop{}
		if errs := tlcsv.LoadRow(ent, row); errs != nil {
			for _, err := range errs {
				panic(err)
			}
		}
		ret = append(ret, ent)
	})
	return ret, err
}

//////////////////

func TestReadTT(t *testing.T) {
	ents, err := ReadTT(TESTFILE)
	assert.NoError(t, err)
	printAll(ents)
}

func BenchmarkReadTT(b *testing.B) {
	for n := 0; n < b.N; n++ {
		a, _ := ReadTT(TESTFILE)
		_ = a
		// printFirst(a)
	}
}

func ReadTT(fn string) ([]any, error) {
	a := tlcsv.NewZipAdapter(fn)
	if err := a.Open(); err != nil {
		panic(err)
	}
	var ret []any
	err := a.ReadRows(TESTTABLE, func(row tlcsv.Row) {
		ent := gtfs.Stop{}
		if errs := tlcsv.LoadRow(&ent, row); errs != nil {
			for _, err := range errs {
				panic(err)
			}
		}
		ret = append(ret, ent)
	})
	return ret, err
}

//////////////////

func TestReadG(t *testing.T) {
	ents, err := ReadG(TESTFILE)
	assert.NoError(t, err)
	printAll(ents)
}

func BenchmarkReadG(b *testing.B) {
	for n := 0; n < b.N; n++ {
		a, _ := ReadG(TESTFILE)
		_ = a
		// printFirst(a)
	}
}

func ReadG(fn string) ([]any, error) {
	a := tlcsv.NewZipAdapter(fn)
	if err := a.Open(); err != nil {
		panic(err)
	}
	var ret []any
	err := a.ReadRows(TESTTABLE, func(row tlcsv.Row) {
		ent := gtfs.Stop{}
		if errs := tlcsv.LoadRow(&ent, row); errs != nil {
			for _, err := range errs {
				panic(err)
			}
		}
		ret = append(ret, ent)
	})
	return ret, err
}
