package gtcsv

import (
	"errors"
	"math"
	"strings"

	"github.com/interline-io/gotransit"
)

// Writer implements a GTFS CSV Writer.
type Writer struct {
	WriterAdapter
	headers map[string][]string
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
func (writer *Writer) NewReader() (gotransit.Reader, error) {
	return NewReader(writer.WriterAdapter.Path())
}

// CopyEntities provides a generic interface for adding Entities.
func (writer *Writer) CopyEntities(ents []gotransit.Entity) error {
	if len(ents) == 0 {
		return nil
	}
	ent := ents[0]
	efn := ents[0].Filename()
	for _, ent := range ents {
		if efn != ent.Filename() {
			return errors.New("all entities must be same type")
		}
	}
	header, ok := writer.headers[efn]
	if !ok {
		h, err := dumpHeader(ent)
		if err != nil {
			return err
		}
		header = h
		writer.headers[efn] = header
		writer.WriterAdapter.WriteRows(efn, [][]string{header})
	}
	rows := [][]string{}
	for _, ent := range ents {
		row, err := dumpRow(ent, header)
		if err != nil {
			return err
		}
		rows = append(rows, row)
	}
	return writer.WriterAdapter.WriteRows(efn, rows)
}

// AddEntity provides a generic interface for adding an Entity.
func (writer *Writer) AddEntity(ent gotransit.Entity) (string, error) {
	if v, ok := ent.(*gotransit.Shape); ok {
		e2s := []gotransit.Entity{}
		es := writer.flattenShape(*v)
		for i := 0; i < len(es); i++ {
			e2s = append(e2s, &es[i])
		}
		return v.EntityID(), writer.CopyEntities(e2s)
	}
	return ent.EntityID(), writer.CopyEntities([]gotransit.Entity{ent})
}

func (writer *Writer) flattenShape(ent gotransit.Shape) []gotransit.Shape {
	coords := ent.Geometry.FlatCoords()
	shapes := []gotransit.Shape{}
	totaldist := 0.0
	for i := 0; i < len(coords); i += 3 {
		s := gotransit.Shape{
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
