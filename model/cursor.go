package model

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type Cursor struct {
	FeedVersionID int
	ID            int
	Valid         bool
}

func NewCursor(fvid int, id int) Cursor {
	return Cursor{FeedVersionID: fvid, ID: id, Valid: true}
}

// UnmarshalJSON implements json.Marshaler interface.
func (r *Cursor) UnmarshalJSON(v []byte) error {
	return r.Scan(v)
}

// MarshalJSON implements the json.marshaler interface.
func (r *Cursor) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.encode())
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (r *Cursor) UnmarshalGQL(v interface{}) error {
	return r.Scan(v)
}

// MarshalGQL implements the graphql.Marshaler interface
func (r Cursor) MarshalGQL(w io.Writer) {
	b, _ := r.MarshalJSON()
	w.Write(b)
}

// UnmarshalGQL implements the graphql.Unmarshaler interface
func (r *Cursor) Scan(value interface{}) error {
	r.Valid = false
	switch v := value.(type) {
	case int64:
		r.ID = int(v)
	case int:
		r.ID = v
	case json.Number:
		a, err := v.Int64()
		if err != nil {
			return err
		}
		r.ID = int(a)
	case []byte:
		return r.decode(string(v))
	case string:
		return r.decode(v)
	case nil:
		// ok
	default:
		return errors.New("invalid cursor")
	}
	r.Valid = true
	return nil
}
func (r *Cursor) encode() string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%d,%d", r.FeedVersionID, r.ID)))
}

func (r *Cursor) decode(value string) error {
	if len(value) == 0 {
		return nil
	}
	dec, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return errors.New("invalid cursor")
	}
	rawSplit := strings.Split(string(dec), ",")
	if len(rawSplit) < 2 {
		return errors.New("invalid cursor")
	}
	r.FeedVersionID, err = strconv.Atoi(rawSplit[0])
	if err != nil {
		return errors.New("invalid cursor")
	}
	r.ID, err = strconv.Atoi(rawSplit[1])
	if err != nil {
		return errors.New("invalid cursor")
	}
	return nil
}
