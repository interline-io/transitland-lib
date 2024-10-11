package empty

import (
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/tt"
)

// Writer is a no-op writer.
type Writer struct {
}

func (*Writer) String() string                          { return "null" }
func (*Writer) Open() error                             { return nil }
func (*Writer) Close() error                            { return nil }
func (*Writer) Create() error                           { return nil }
func (*Writer) Delete() error                           { return nil }
func (*Writer) NewReader() (adapters.Reader, error)     { return nil, nil }
func (*Writer) AddEntity(ent tt.Entity) (string, error) { return ent.EntityID(), nil }
func (*Writer) AddEntities(ents []tt.Entity) ([]string, error) {
	retids := []string{}
	for _, ent := range ents {
		retids = append(retids, ent.EntityID())
	}
	return retids, nil
}
