package tlcsv

import (
	"errors"
	"math"
	"strings"

	"github.com/interline-io/transitland-lib/tl"
)

type hasEntityKey interface {
	EntityKey() string
}

// Writer implements a GTFS CSV Writer.
type Writer struct {
	WriterAdapter
	entHeaders map[string][]string
	extHeaders map[string][]string
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
		entHeaders:    map[string][]string{},
		extHeaders:    map[string][]string{},
	}, nil
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
func (writer *Writer) NewReader() (tl.Reader, error) {
	return NewReader(writer.WriterAdapter.Path())
}

// AddEntities writes entities to the output.
func (writer *Writer) AddEntities(ents []tl.Entity) ([]string, error) {
	eids := []string{}
	if len(ents) == 0 {
		return eids, nil
	}
	ent := ents[0]
	efn := ents[0].Filename()
	for _, ent := range ents {
		if efn != ent.Filename() {
			return eids, errors.New("all entities must be same type")
		}
	}
	extHeader := writer.extHeaders[efn]
	entHeader, ok := writer.entHeaders[efn]
	if !ok {
		h, err := dumpHeader(ent)
		if err != nil {
			return eids, err
		}
		entHeader = h
		for k := range ent.Extra() {
			extHeader = append(extHeader, k)
		}
		writer.entHeaders[efn] = entHeader
		writer.extHeaders[efn] = extHeader
		fileHeader := []string{}
		fileHeader = append(fileHeader, entHeader...)
		fileHeader = append(fileHeader, extHeader...)
		writer.WriterAdapter.WriteRows(efn, [][]string{fileHeader})
	}
	rows := [][]string{}
	for _, ent := range ents {
		sid := ""
		if v, ok := ent.(hasEntityKey); ok {
			sid = v.EntityKey()
		}
		row, err := dumpRow(ent, entHeader)
		if err != nil {
			return eids, err
		}
		for _, k := range extHeader {
			eh, _ := ent.GetExtra(k)
			row = append(row, eh)
		}
		rows = append(rows, row)
		eids = append(eids, sid)
	}
	err := writer.WriterAdapter.WriteRows(efn, rows)
	return eids, err
}

// AddEntity writes an entity to the output.
func (writer *Writer) AddEntity(ent tl.Entity) (string, error) {
	eids := []string{}
	var err error
	if v, ok := ent.(*tl.Shape); ok {
		e2s := []tl.Entity{}
		es := writer.flattenShape(*v)
		for i := 0; i < len(es); i++ {
			e2s = append(e2s, &es[i])
		}
		eids, err = writer.AddEntities(e2s)
	} else {
		eids, err = writer.AddEntities([]tl.Entity{ent})
	}
	if err != nil {
		return "", err
	}
	if len(eids) == 0 {
		return "", errors.New("did not write expected number of entities")
	}
	return eids[0], nil
}

func (writer *Writer) flattenShape(ent tl.Shape) []tl.Shape {
	coords := ent.Geometry.FlatCoords()
	shapes := []tl.Shape{}
	totaldist := 0.0
	for i := 0; i < len(coords); i += 3 {
		s := tl.Shape{
			ShapeID:           ent.ShapeID,
			ShapePtSequence:   i,
			ShapePtLon:        coords[i],
			ShapePtLat:        coords[i+1],
			ShapeDistTraveled: coords[i+2],
		}
		totaldist += coords[i+2]
		shapes = append(shapes, s)
	}
	// Set any zeros to NaN
	cur := 0.0
	for i := 0; i < len(shapes); i++ {
		if shapes[i].ShapeDistTraveled < cur {
			shapes[i].ShapeDistTraveled = math.NaN()
		}
		cur = shapes[i].ShapeDistTraveled
	}
	if cur == 0.0 {
		for i := 0; i < len(shapes); i++ {
			shapes[i].ShapeDistTraveled = math.NaN()
		}
	}
	return shapes
}
