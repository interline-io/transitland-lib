package tt

import (
	"encoding/json"
	"io"
)

// Fixes issue with gqlgen and generics

func (r *String) UnmarshalGQL(v interface{}) error { return r.Scan(v) }
func (r *Int) UnmarshalGQL(v interface{}) error    { return r.Scan(v) }
func (r *Key) UnmarshalGQL(v interface{}) error    { return r.Scan(v) }

// func (r *Float) UnmarshalGQL(v interface{}) error  { return r.Scan(v) }
// func (r *Date) UnmarshalGQL(v interface{}) error   { return r.Scan(v) }
// func (r *Time) UnmarshalGQL(v interface{}) error   { return r.Scan(v) }

func (r String) MarshalGQL(w io.Writer) { w.Write(gqlWrite(r)) }
func (r Int) MarshalGQL(w io.Writer)    { w.Write(gqlWrite(r)) }

// func (r Float) MarshalGQL(w io.Writer) { w.Write(gqlWrite(r)) }
// func (r Date) MarshalGQL(w io.Writer)  { w.Write(gqlWrite(r)) }
// func (r Time) MarshalGQL(w io.Writer)  { w.Write(gqlWrite(r)) }

func gqlWrite(a json.Marshaler) []byte {
	b, _ := a.MarshalJSON()
	return b
}
