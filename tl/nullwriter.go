package tl

// NullWriter is a no-op writer.
type NullWriter struct {
}

func (*NullWriter) String() string                       { return "null" }
func (*NullWriter) Open() error                          { return nil }
func (*NullWriter) Close() error                         { return nil }
func (*NullWriter) Create() error                        { return nil }
func (*NullWriter) Delete() error                        { return nil }
func (*NullWriter) NewReader() (Reader, error)           { return nil, nil }
func (*NullWriter) AddEntity(ent Entity) (string, error) { return ent.EntityID(), nil }
func (*NullWriter) AddEntities(ents []Entity) ([]string, error) {
	retids := []string{}
	for _, ent := range ents {
		retids = append(retids, ent.EntityID())
	}
	return retids, nil
}
