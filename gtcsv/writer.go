package gtcsv

import (
	"errors"
	"math"
	"strings"

	"github.com/interline-io/gotransit"
)

// Writer implements a GTFS CSV Writer.
type Writer struct {
	Adapter WriterAdapter
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
		Adapter: a,
		headers: map[string][]string{},
	}, nil
}

// Open the Writer.
func (writer *Writer) Open() error {
	return writer.Adapter.Open()
}

// Close the Writer.
func (writer *Writer) Close() error {
	return writer.Adapter.Close()
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
	return NewReader(writer.Adapter.Path())
}

// AddEntities provides a generic interface for adding Entities.
func (writer *Writer) AddEntities(ents []gotransit.Entity) error {
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
		writer.Adapter.WriteRows(efn, [][]string{header})
	}
	rows := [][]string{}
	for _, ent := range ents {
		row, err := dumpRow(ent, header)
		if err != nil {
			return err
		}
		rows = append(rows, row)
	}
	return writer.Adapter.WriteRows(efn, rows)
}

// AddEntity provides a generic interface for adding an Entity.
func (writer *Writer) AddEntity(ent gotransit.Entity) (string, error) {
	if v, ok := ent.(*gotransit.Shape); ok {
		e2s := []gotransit.Entity{}
		for _, s := range writer.flattenShape(*v) {
			e2s = append(e2s, &s)
		}
		return v.EntityID(), writer.AddEntities(e2s)
	}
	return ent.EntityID(), writer.AddEntities([]gotransit.Entity{ent})
}

func (writer *Writer) flattenShape(ent gotransit.Shape) []gotransit.Shape {
	coords := ent.Geometry.FlatCoords()
	seq := 1
	shapes := []gotransit.Shape{}
	totaldist := 0.0
	for i := 0; i < len(coords); i += 3 {
		s := gotransit.Shape{
			ShapeID:           ent.ShapeID,
			ShapePtSequence:   seq,
			ShapePtLon:        coords[i],
			ShapePtLat:        coords[i+1],
			ShapeDistTraveled: coords[i+2],
		}
		totaldist += coords[i+2]
		seq++
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
