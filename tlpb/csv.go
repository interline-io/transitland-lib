package tlpb

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl/tt"
	"github.com/interline-io/transitland-lib/tlcsv"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func (t *Url) FromCsv(v string) error {
	t.Url = v
	return nil
}

func (t *EntityID) FromCsv(v string) error {
	t.Id = v
	return nil
}

func (t *Reference) FromCsv(v string) error {
	t.EntityId = v
	return nil
}

func (t *Seconds) FromCsv(v string) error {
	wt, err := tt.NewWideTime(v)
	if err != nil {
		return err
	}
	t.Seconds = int64(wt.Seconds)
	return nil
}

func pbJson(v protoreflect.ProtoMessage) string {
	jj, _ := protojson.Marshal(v)
	return string(jj)
}

func ReadStops(fn string) error {
	a := tlcsv.NewZipAdapter(fn)
	if err := a.Open(); err != nil {
		panic(err)
	}
	a.ReadRows("stop_times.txt", func(row tlcsv.Row) {
		ent := StopTime{}
		if errs := tlcsv.LoadRow(&ent, row); errs != nil {
			for _, err := range errs {
				panic(err)
			}
		}
		fmt.Println(pbJson(&ent))
	})
	return nil
}
