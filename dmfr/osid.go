package dmfr

import (
	"github.com/interline-io/transitland-lib/adapters/empty"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tl"
)

type EntityOnestopID struct {
	Filename  string
	EntityID  string
	OnestopID string
}

func NewOnestopIDsFromReader(reader tl.Reader) ([]EntityOnestopID, error) {
	cp, err := copier.NewCopier(reader, &empty.Writer{}, copier.Options{})
	if err != nil {
		return nil, err
	}
	cpResult := cp.Copy()
	if cpResult.WriteError != nil {
		return nil, cpResult.WriteError
	}
	_ = cpResult

	return nil, nil
}
