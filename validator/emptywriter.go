package validator

import "github.com/interline-io/gotransit"

type emptyWriter struct {
}

func (*emptyWriter) Open() error                                 { return nil }
func (*emptyWriter) Close() error                                { return nil }
func (*emptyWriter) Create() error                               { return nil }
func (*emptyWriter) Delete() error                               { return nil }
func (*emptyWriter) NewReader() (gotransit.Reader, error)           { return nil, nil }
func (*emptyWriter) AddEntity(ent gotransit.Entity) (string, error) { return ent.EntityID(), nil }
func (*emptyWriter) AddEntities(ents []gotransit.Entity) error      { return nil }
