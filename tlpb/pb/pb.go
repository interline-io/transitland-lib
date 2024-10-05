//go:generate protoc --go_out=. --go_opt=paths=source_relative --go_opt=Mgtfs.proto=tlpb/pb -I .. ../gtfs.proto

package pb

import "github.com/interline-io/transitland-lib/tl/tt"

func (t *String) FromCsv(v string) error {
	t.Val = v
	return nil
}

func (t *Int) FromCsv(v string) error {
	t.Val = 123
	return nil
}

func (t *Float) FromCsv(v string) error {
	t.Val = 123.0
	return nil
}

func (t *Url) FromCsv(v string) error {
	t.Val = v
	return nil
}

func (t *Key) FromCsv(v string) error {
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

func (t *StopLocationType) FromCsv(v string) error {
	t.Val = 123
	return nil
}

func (t *WheelchairAccess) FromCsv(v string) error {
	t.Val = 123
	return nil
}

func (t *StopTimepoint) FromCsv(v string) error {
	t.Val = 123
	return nil
}
