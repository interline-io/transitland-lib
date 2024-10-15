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

// AddEntities writes entities to the output.
func (writer *Writer) AddEntities(ents []tt.Entity) ([]string, error) {
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
		// Horrible special case bug fix
		if v, ok := ent.(*gtfs.Stop); ok {
			c := v.Coordinates()
			v.StopLon.Set(c[0])
			v.StopLat.Set(c[1])
		}
	}
	header, ok := writer.headers[efn]
	extraHeader, ok := writer.extraHeaders[efn]
	if !ok {
		h, err := dumpHeader(ent)
		if err != nil {
			return eids, err
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
			return eids, err
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

// AddEntity writes an entity to the output.
func (writer *Writer) AddEntity(ent tt.Entity) (string, error) {
	eids := []string{}
	var err error
	if v, ok := ent.(*gtfs.Shape); ok {
		e2s := []tt.Entity{}
		es := writer.flattenShape(*v)
		for i := 0; i < len(es); i++ {
			e2s = append(e2s, &es[i])
		}
		eids, err = writer.AddEntities(e2s)
	} else {
		eids, err = writer.AddEntities([]tt.Entity{ent})
	}
	if err != nil {
		return "", err
	}
	if len(eids) == 0 {
		return "", errors.New("did not write expected number of entities")
	}
	return eids[0], nil
}

func (writer *Writer) flattenShape(ent gtfs.Shape) []gtfs.Shape {
	coords := ent.Geometry.FlatCoords()
	shapes := []gtfs.Shape{}
	totaldist := 0.0
	for i := 0; i < len(coords); i += 3 {
		s := gtfs.Shape{
			ShapeID:           ent.ShapeID,
			ShapePtSequence:   tt.NewInt(i),
			ShapePtLon:        tt.NewFloat(coords[i]),
			ShapePtLat:        tt.NewFloat(coords[i+1]),
			ShapeDistTraveled: tt.NewFloat(coords[i+2]),
		}
		totaldist += coords[i+2]
		shapes = append(shapes, s)
	}
	cur := 0.0
	for i := 0; i < len(shapes); i++ {
		if shapes[i].ShapeDistTraveled.Val < cur {
			shapes[i].ShapeDistTraveled.Unset()
		}
		cur = shapes[i].ShapeDistTraveled.Val
	}
	if cur == 0.0 {
		for i := 0; i < len(shapes); i++ {
			shapes[i].ShapeDistTraveled.Unset()
		}
	}
	return shapes
}
