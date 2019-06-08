package gtcsv

import (
	"errors"
	"math"
	"os"

	"github.com/interline-io/gotransit"
)

// Writer implements a GTFS CSV Writer.
type Writer struct {
	Adapter WriterAdapter
	headers map[string][]string
	files   map[string]*os.File
}

// NewWriter returns a new Writer.
func NewWriter(path string) (*Writer, error) {
	a := NewZipWriterAdapter(path)
	if a == nil {
		return nil, errors.New("no adapter found")
	}
	return &Writer{
		Adapter: a,
		headers: map[string][]string{},
		files:   map[string]*os.File{},
	}, nil
}

// Open the Writer.
func (writer *Writer) Open() error {
	return nil
}

// Close the Writer.
func (writer *Writer) Close() error {
	return writer.Adapter.Close()
}

// Create the necessary files for the Writer.
func (writer *Writer) Create() error {
	return writer.Adapter.Create()
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
	stoptimes := []*gotransit.StopTime{}
	for _, ent := range ents {
		if v, ok := ent.(*gotransit.StopTime); ok {
			stoptimes = append(stoptimes, v)
		}
	}

	for i := 0; i < len(stoptimes); i++ {
		if stoptimes[i].ArrivalTime == 0 {
			stoptimes[i].ArrivalTime = math.MaxInt64
		}
		if stoptimes[i].DepartureTime == 0 {
			stoptimes[i].DepartureTime = math.MaxInt64
		}
	}

	// Set any zeros to NaN
	cur := stoptimes[0].ShapeDistTraveled
	for i := 1; i < len(stoptimes); i++ {
		if stoptimes[i].ShapeDistTraveled <= cur {
			stoptimes[i].ShapeDistTraveled = math.NaN()
		} else {
			cur = stoptimes[i].ShapeDistTraveled
		}
	}
	if cur == 0.0 {
		for i := 0; i < len(stoptimes); i++ {
			stoptimes[i].ShapeDistTraveled = math.NaN()
		}
	}

	for _, ent := range ents {
		writer.AddEntity(ent)
	}
	return nil
}

// AddEntity provides a generic interface for adding an Entity.
func (writer *Writer) AddEntity(ent gotransit.Entity) (string, error) {
	switch v := ent.(type) {
	case *gotransit.Shape:
		var eid string
		var err error
		for _, s := range writer.flattenShape(*v) {
			eid, err = writer.addEntity(&s)
		}
		return eid, err
	default:
		return writer.addEntity(ent)
	}
}

func (writer *Writer) addEntity(ent gotransit.Entity) (string, error) {
	// Is this file open
	efn := ent.Filename()
	header, ok := writer.headers[efn]
	if !ok {
		h, err := dumpHeader(ent)
		if err != nil {
			return "", err
		}
		header = h
		writer.headers[efn] = header
		writer.Adapter.WriteRow(efn, header)
	}
	row, err := dumpRow(ent, header)
	if err != nil {
		return "", err
	}
	if err := writer.Adapter.WriteRow(efn, row); err != nil {
		panic(err)
	}
	return ent.EntityID(), nil
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

func boolToInt(b bool) int {
	if b == false {
		return 0
	}
	return 1
}
