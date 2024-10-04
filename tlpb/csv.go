package tlpb

import (
	"encoding/json"
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
	"github.com/interline-io/transitland-lib/tlcsv"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func (t *Url) FromCsv(v string) error {
	t.Val = v
	return nil
}

func (t *EntityID) FromCsv(v string) error {
	t.Val = v
	return nil
}

func (t *Reference) FromCsv(v string) error {
	t.Val = v
	return nil
}

func (t *Seconds) FromCsv(v string) error {
	wt, err := tt.NewWideTime(v)
	if err != nil {
		return err
	}
	t.Val = int64(wt.Seconds)
	return nil
}

func pbJson(v protoreflect.ProtoMessage) string {
	jj, _ := protojson.Marshal(v)
	return string(jj)
}

func ReadStopsPB(fn string) error {
	a := tlcsv.NewZipAdapter(fn)
	if err := a.Open(); err != nil {
		panic(err)
	}
	a.ReadRows("stops.txt", func(row tlcsv.Row) {
		ent := Stop{}
		if errs := tlcsv.LoadRow(&ent, row); errs != nil {
			for _, err := range errs {
				panic(err)
			}
		}
		_ = ent
		fmt.Println(pbJson(&ent))
	})
	return nil
}

func ReadStopsTT(fn string) error {
	a := tlcsv.NewZipAdapter(fn)
	if err := a.Open(); err != nil {
		panic(err)
	}
	a.ReadRows("stops.txt", func(row tlcsv.Row) {
		ent := tl.Stop{}
		if errs := tlcsv.LoadRow(&ent, row); errs != nil {
			for _, err := range errs {
				panic(err)
			}
		}
		_ = ent
		jj, _ := json.Marshal(&ent)
		fmt.Println(string(jj))
		// fmt.Println(pbJson(&ent))
	})
	return nil
}
