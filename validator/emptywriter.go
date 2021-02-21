package validator

import "github.com/interline-io/transitland-lib/tl"

type emptyWriter struct {
}

func (*emptyWriter) Open() error                             { return nil }
func (*emptyWriter) Close() error                            { return nil }
func (*emptyWriter) Create() error                           { return nil }
func (*emptyWriter) Delete() error                           { return nil }
func (*emptyWriter) NewReader() (tl.Reader, error)           { return nil, nil }
func (*emptyWriter) AddEntity(ent tl.Entity) (string, error) { return ent.EntityID(), nil }
func (*emptyWriter) AddEntities(ents []tl.Entity) ([]string, error) {
	retids := []string{}
	for _, ent := range ents {
		retids = append(retids, ent.EntityID())
	}
	return retids, nil
}
