package tt

import (
	"io"
)

// Option[T] types need MarshalGQL / UnmarshalGQL defined directly

func (r Bool) MarshalGQL(w io.Writer)       { marshalGql(r, w) }
func (r Date) MarshalGQL(w io.Writer)       { marshalGql(r, w) }
func (r Float) MarshalGQL(w io.Writer)      { marshalGql(r, w) }
func (r Geometry) MarshalGQL(w io.Writer)   { marshalGql(r, w) }
func (r Int) MarshalGQL(w io.Writer)        { marshalGql(r, w) }
func (r LineString) MarshalGQL(w io.Writer) { marshalGql(r, w) }
func (r Map) MarshalGQL(w io.Writer)        { marshalGql(r, w) }
func (r Point) MarshalGQL(w io.Writer)      { marshalGql(r, w) }
func (r Polygon) MarshalGQL(w io.Writer)    { marshalGql(r, w) }
func (r Seconds) MarshalGQL(w io.Writer)    { marshalGql(r, w) }
func (r String) MarshalGQL(w io.Writer)     { marshalGql(r, w) }
func (r Strings) MarshalGQL(w io.Writer)    { marshalGql(r, w) }
func (r Tags) MarshalGQL(w io.Writer)       { marshalGql(r, w) }
func (r Time) MarshalGQL(w io.Writer)       { marshalGql(r, w) }

func (r *Bool) UnmarshalGQL(v any) error       { return unmarshalGql(r, v) }
func (r *Date) UnmarshalGQL(v any) error       { return unmarshalGql(r, v) }
func (r *Float) UnmarshalGQL(v any) error      { return unmarshalGql(r, v) }
func (r *Geometry) UnmarshalGQL(v any) error   { return unmarshalGql(r, v) }
func (r *Int) UnmarshalGQL(v any) error        { return unmarshalGql(r, v) }
func (r *LineString) UnmarshalGQL(v any) error { return unmarshalGql(r, v) }
func (r *Map) UnmarshalGQL(v any) error        { return unmarshalGql(r, v) }
func (r *Point) UnmarshalGQL(v any) error      { return unmarshalGql(r, v) }
func (r *Polygon) UnmarshalGQL(v any) error    { return unmarshalGql(r, v) }
func (r *Seconds) UnmarshalGQL(v any) error    { return unmarshalGql(r, v) }
func (r *String) UnmarshalGQL(v any) error     { return unmarshalGql(r, v) }
func (r *Strings) UnmarshalGQL(v any) error    { return unmarshalGql(r, v) }
func (r *Tags) UnmarshalGQL(v any) error       { return unmarshalGql(r, v) }
func (r *Time) UnmarshalGQL(v any) error       { return unmarshalGql(r, v) }

type canUnmarshalJson interface {
	Scan(any) error
}

type canMarshalJson interface {
	MarshalJSON() ([]byte, error)
}

func unmarshalGql(v canUnmarshalJson, d any) error {
	return v.Scan(d)
}

func marshalGql(v canMarshalJson, w io.Writer) {
	b, _ := v.MarshalJSON()
	w.Write(b)
}
