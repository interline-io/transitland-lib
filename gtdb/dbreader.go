package gtdb

import (
	"reflect"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
	"github.com/jinzhu/gorm"
)

// Reader reads from a database.
type Reader struct {
	Adapter       Adapter
	PageSize      int
	FeedVersionID int
}

// NewReader returns an initialized Reader.
func NewReader(dburl string) (*Reader, error) {
	return &Reader{Adapter: NewAdapter(dburl), PageSize: 1000}, nil
}

// ValidateStructure returns if all the necessary tables are present. Not implemented.
func (reader *Reader) ValidateStructure() []error {
	errs := []error{}
	return errs
}

// Open the database.
func (reader *Reader) Open() error {
	return reader.Adapter.Open()
}

// Close the database.
func (reader *Reader) Close() error {
	return reader.Adapter.Close()
}

// Where returns a GORM DB with default Where clause set.
func (reader *Reader) Where() *gorm.DB {
	db := reader.Adapter.DB()
	if reader.FeedVersionID > 0 {
		db = db.Where("feed_version_id = ?", reader.FeedVersionID)
	}
	return db
}

// ReadEntities provides a generic interface for reading Entities.
func (reader *Reader) ReadEntities(c interface{}) error {
	// Seems to work.
	outValue := reflect.ValueOf(c)
	outInnerType := outValue.Type().Elem()
	outInner := reflect.New(outInnerType)
	ent, ok := outInner.Interface().(gotransit.Entity)
	if !ok {
		return causes.NewSourceUnreadableError("not an entity", nil)
	}
	_ = ent
	slice := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(outInner.Interface())), 0, 0)
	// Create a pointer to a slice value and set it to the slice
	x := reflect.New(slice.Type())
	z := x.Elem()
	z.Set(slice)
	//
	db := reader.Adapter.DB()
	db.Find(x.Interface())
	go func() {
		for i := 0; i < z.Len(); i++ {
			p := z.Index(i)
			outValue.Send(p.Elem())
		}
		outValue.Close()
	}()
	return nil
}

// ShapeLinesByShapeID sends single-geometry LineString Shapes
func (reader *Reader) ShapeLinesByShapeID(shapeIDs ...string) chan gotransit.Shape {
	out := make(chan gotransit.Shape, bufferSize)
	go func() {
		for shapes := range reader.ShapesByShapeID(shapeIDs...) {
			out <- shapes[0]
		}
		close(out)
	}()
	return out
}

// ShapesByShapeID sends shapes grouped by ShapeID.
func (reader *Reader) ShapesByShapeID(shapeIDs ...string) chan []gotransit.Shape {
	if len(shapeIDs) == 0 {
		rows, err := reader.Where().Model(&gotransit.Shape{}).Select("distinct(id)").Rows()
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		for rows.Next() {
			shapeID := ""
			rows.Scan(&shapeID)
			shapeIDs = append(shapeIDs, shapeID)
		}
	}
	out := make(chan []gotransit.Shape, bufferSize)
	go func() {
		for _, shapeID := range shapeIDs {
			ents := []gotransit.Shape{}
			reader.Where().Where("id = ?", shapeID).Find(&ents)
			if len(ents) > 0 {
				out <- ents
			}
		}
		close(out)
	}()
	return out
}

// StopTimesByTripID sends StopTimes grouped by TripID.
func (reader *Reader) StopTimesByTripID(tripIDs ...string) chan []gotransit.StopTime {
	if len(tripIDs) == 0 {
		rows, err := reader.Where().Model(&gotransit.Trip{}).Select("id").Rows()
		if err != nil {
			panic(err)
		}
		defer rows.Close()
		for rows.Next() {
			tripID := ""
			rows.Scan(&tripID)
			tripIDs = append(tripIDs, tripID)
		}
	}
	out := make(chan []gotransit.StopTime, bufferSize)
	go func() {
		for _, tripID := range tripIDs {
			ents := []gotransit.StopTime{}
			reader.Where().Where("trip_id = ?", tripID).Find(&ents)
			if len(ents) > 0 {
				out <- ents
			}
		}
		close(out)
	}()
	return out
}

// Stops sends Stops.
func (reader *Reader) Stops() chan gotransit.Stop {
	out := make(chan gotransit.Stop, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []gotransit.Stop{}
			reader.Where().Order("id").Offset(offset).Limit(reader.PageSize).Find(&ents)
			for _, ent := range ents {
				out <- ent
			}
			if len(ents) < reader.PageSize {
				break
			}
			offset = offset + reader.PageSize
		}
		close(out)
	}()
	return out
}

// StopTimes sends StopTimes.
func (reader *Reader) StopTimes() chan gotransit.StopTime {
	out := make(chan gotransit.StopTime, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []gotransit.StopTime{}
			reader.Where().Order("id").Offset(offset).Limit(reader.PageSize).Find(&ents)
			for _, ent := range ents {
				out <- ent
			}
			if len(ents) < reader.PageSize {
				break
			}
			offset = offset + reader.PageSize
		}
		close(out)
	}()
	return out
}

// Agencies sends Agencies.
func (reader *Reader) Agencies() chan gotransit.Agency {
	out := make(chan gotransit.Agency, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []gotransit.Agency{}
			reader.Where().Order("id").Offset(offset).Limit(reader.PageSize).Find(&ents)
			for _, ent := range ents {
				out <- ent
			}
			if len(ents) < reader.PageSize {
				break
			}
			offset = offset + reader.PageSize
		}
		close(out)
	}()
	return out
}

// Calendars sends Calendars.
func (reader *Reader) Calendars() chan gotransit.Calendar {
	out := make(chan gotransit.Calendar, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []gotransit.Calendar{}
			reader.Where().Order("id").Offset(offset).Limit(reader.PageSize).Find(&ents)
			for _, ent := range ents {
				out <- ent
			}
			if len(ents) < reader.PageSize {
				break
			}
			offset = offset + reader.PageSize
		}
		close(out)
	}()
	return out
}

// CalendarDates sends CalendarDates.
func (reader *Reader) CalendarDates() chan gotransit.CalendarDate {
	out := make(chan gotransit.CalendarDate, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []gotransit.CalendarDate{}
			reader.Where().Order("id").Offset(offset).Limit(reader.PageSize).Find(&ents)
			for _, ent := range ents {
				out <- ent
			}
			if len(ents) < reader.PageSize {
				break
			}
			offset = offset + reader.PageSize
		}
		close(out)
	}()
	return out
}

// FareAttributes sends FareAttributes.
func (reader *Reader) FareAttributes() chan gotransit.FareAttribute {
	out := make(chan gotransit.FareAttribute, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []gotransit.FareAttribute{}
			reader.Where().Order("id").Offset(offset).Limit(reader.PageSize).Find(&ents)
			for _, ent := range ents {
				out <- ent
			}
			if len(ents) < reader.PageSize {
				break
			}
			offset = offset + reader.PageSize
		}
		close(out)
	}()
	return out
}

// FareRules sends FareRules.
func (reader *Reader) FareRules() chan gotransit.FareRule {
	out := make(chan gotransit.FareRule, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []gotransit.FareRule{}
			reader.Where().Order("id").Offset(offset).Limit(reader.PageSize).Find(&ents)
			for _, ent := range ents {
				out <- ent
			}
			if len(ents) < reader.PageSize {
				break
			}
			offset = offset + reader.PageSize
		}
		close(out)
	}()
	return out
}

// FeedInfos sends FeedInfos.
func (reader *Reader) FeedInfos() chan gotransit.FeedInfo {
	out := make(chan gotransit.FeedInfo, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []gotransit.FeedInfo{}
			reader.Where().Order("id").Offset(offset).Limit(reader.PageSize).Find(&ents)
			for _, ent := range ents {
				out <- ent
			}
			if len(ents) < reader.PageSize {
				break
			}
			offset = offset + reader.PageSize
		}
		close(out)
	}()
	return out
}

// Frequencies sends Frequencies.
func (reader *Reader) Frequencies() chan gotransit.Frequency {
	out := make(chan gotransit.Frequency, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []gotransit.Frequency{}
			reader.Where().Order("id").Offset(offset).Limit(reader.PageSize).Find(&ents)
			for _, ent := range ents {
				out <- ent
			}
			if len(ents) < reader.PageSize {
				break
			}
			offset = offset + reader.PageSize
		}
		close(out)
	}()
	return out
}

// Routes sends Routes.
func (reader *Reader) Routes() chan gotransit.Route {
	out := make(chan gotransit.Route, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []gotransit.Route{}
			reader.Where().Order("id").Offset(offset).Limit(reader.PageSize).Find(&ents)
			for _, ent := range ents {
				out <- ent
			}
			if len(ents) < reader.PageSize {
				break
			}
			offset = offset + reader.PageSize
		}
		close(out)
	}()
	return out
}

// Shapes sends Shapes.
func (reader *Reader) Shapes() chan gotransit.Shape {
	out := make(chan gotransit.Shape, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []gotransit.Shape{}
			reader.Where().Order("id").Offset(offset).Limit(reader.PageSize).Find(&ents)
			for _, ent := range ents {
				out <- ent
			}
			if len(ents) < reader.PageSize {
				break
			}
			offset = offset + reader.PageSize
		}
		close(out)
	}()
	return out
}

// Transfers sends Transfers.
func (reader *Reader) Transfers() chan gotransit.Transfer {
	out := make(chan gotransit.Transfer, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []gotransit.Transfer{}
			reader.Where().Order("id").Offset(offset).Limit(reader.PageSize).Find(&ents)
			for _, ent := range ents {
				out <- ent
			}
			if len(ents) < reader.PageSize {
				break
			}
			offset = offset + reader.PageSize
		}
		close(out)
	}()
	return out
}

// Trips sends Trips.
func (reader *Reader) Trips() chan gotransit.Trip {
	out := make(chan gotransit.Trip, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []gotransit.Trip{}
			reader.Where().Order("id").Offset(offset).Limit(reader.PageSize).Find(&ents)
			for _, ent := range ents {
				out <- ent
			}
			if len(ents) < reader.PageSize {
				break
			}
			offset = offset + reader.PageSize
		}
		close(out)
	}()
	return out
}
