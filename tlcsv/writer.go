package tlcsv

import (
	"errors"
	"strings"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

type hasEntityKey interface {
	EntityKey() string
}

// Writer implements a GTFS CSV Writer.
type Writer struct {
	WriterAdapter
	writeExtraColumns bool
	headers           map[string][]string
	extraHeaders      map[string][]string
}

// NewWriter returns a new Writer.
func NewWriter(path string) (*Writer, error) {
	var a WriterAdapter
	if strings.HasSuffix(path, ".zip") {
		a = NewZipWriterAdapter(path)
	} else {
		a = NewDirAdapter(path)
	}
	return &Writer{
		WriterAdapter: a,
		headers:       map[string][]string{},
		extraHeaders:  map[string][]string{},
	}, nil
}

func (writer *Writer) WriteExtraColumns(val bool) {
	writer.writeExtraColumns = val
}

func (writer *Writer) String() string {
	return writer.WriterAdapter.Path()
}

// Create the necessary files for the Writer.
func (writer *Writer) Create() error {
	// TODO: return error when output path exists
	return nil
}

// Delete the Writer.
func (writer *Writer) Delete() error {
	return nil
}

// NewReader returns a new Reader for the Writer destination.
func (writer *Writer) NewReader() (adapters.Reader, error) {
	return NewReader(writer.WriterAdapter.Path())
}

// AddEntity writes an entity to the output.
func (writer *Writer) AddEntity(ent tt.Entity) (string, error) {
	eids, err := writer.AddEntities([]tt.Entity{ent})
	if err != nil {
		return "", err
	}
	if len(eids) == 0 {
		return "", errors.New("did not write expected number of entities")
	}
	return eids[0], nil
}

// AddEntities writes entities to the output.
func (writer *Writer) AddEntities(ents []tt.Entity) ([]string, error) {
	if len(ents) == 0 {
		return nil, nil
	}

	// Awful Ugly Hack to Flatten entities
	var expandedEnts []tt.Entity
	var originalEids []string
	for _, ent := range ents {
		if a, ok := ent.(canFlatten); ok {
			eid := ""
			if b, ok := ent.(hasEntityKey); ok {
				eid = b.EntityKey()
			}
			originalEids = append(originalEids, eid)
			expandedEnts = append(expandedEnts, a.Flatten()...)
		}
	}
	if len(expandedEnts) > 0 {
		_, err := writer.addBatch(expandedEnts)
		if err != nil {
			return nil, err
		}
		return originalEids, nil
	}

	// Normal write path
	return writer.addBatch(ents)
}

func (writer *Writer) addBatch(ents []tt.Entity) ([]string, error) {
	var eids []string
	if len(ents) == 0 {
		return nil, nil
	}

	ent := ents[0]
	efn := ents[0].Filename()
	for _, ent := range ents {
		if efn != ent.Filename() {
			return nil, errors.New("all entities must be same type")
		}
		// Horrible special case bug fix
		if v, ok := ent.(*gtfs.Stop); ok {
			c := v.Coordinates()
			v.StopLon.Set(c[0])
			v.StopLat.Set(c[1])
		}
	}

	extraHeader := writer.extraHeaders[efn]
	header, ok := writer.headers[efn]
	if !ok {
		h, err := dumpHeader(ent)
		if err != nil {
			return nil, err
		}
		header = h
		if extEnt, ok2 := ent.(tt.EntityWithExtra); ok2 && writer.writeExtraColumns {
			extraHeader = extEnt.ExtraKeys()
		}
		writer.headers[efn] = header
		writer.extraHeaders[efn] = extraHeader
		h2 := append([]string{}, header...)
		h2 = append(h2, extraHeader...)
		writer.WriterAdapter.WriteRows(efn, [][]string{h2})
	}
	rows := [][]string{}
	for _, ent := range ents {
		sid := ""
		if v, ok := ent.(hasEntityKey); ok {
			sid = v.EntityKey()
		}
		row, err := dumpRow(ent, header)
		if err != nil {
			return nil, err
		}
		if len(extraHeader) > 0 {
			if extEnt, ok := ent.(tt.EntityWithExtra); ok {
				for _, extraKey := range extraHeader {
					a, _ := extEnt.GetExtra(extraKey)
					row = append(row, a)
				}

			}
		}
		rows = append(rows, row)
		eids = append(eids, sid)
	}
	err := writer.WriterAdapter.WriteRows(efn, rows)
	return eids, err
}

type canFlatten interface {
	Flatten() []tt.Entity
}
