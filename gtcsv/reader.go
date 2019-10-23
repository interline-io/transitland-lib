package gtcsv

import (
	"encoding/csv"
	"errors"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
	"github.com/interline-io/gotransit/internal/tags"

	"github.com/dimchansky/utfbom"
)

// s2D is two dimensional string slice
type s2D = [][]string

// Reader reads GTFS entities from CSV files.
type Reader struct {
	Adapter Adapter
}

// NewReader returns an initialized CSV Reader.
func NewReader(path string) (*Reader, error) {
	var a Adapter
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		a = &URLAdapter{url: path}
	} else if strings.HasSuffix(path, ".zip") {
		a = &ZipAdapter{path: path}
	} else {
		a = NewDirAdapter(path)
	}
	return &Reader{Adapter: a}, nil
}

// Open the source for reading.
func (reader *Reader) Open() error {
	return reader.Adapter.Open()
}

// Close the source.
func (reader *Reader) Close() error {
	return reader.Adapter.Close()
}

// ReadEntities provides a generic interface for reading Entities.
func (reader *Reader) ReadEntities(c interface{}) error {
	// Magic
	outValue := reflect.ValueOf(c)
	outInnerType := outValue.Type().Elem()
	outInner := reflect.New(outInnerType)
	ent, ok := outInner.Interface().(gotransit.Entity)
	if !ok {
		return causes.NewSourceUnreadableError("not a valid entity", nil)
	}
	reader.Adapter.ReadRows(ent.Filename(), func(row Row) {
		a := reflect.New(outInnerType)
		e := a.Interface().(gotransit.Entity)
		loadRow(e, row)
		outValue.Send(a.Elem())
	})
	outValue.Close()
	return nil
}

// ValidateStructure returns if all required CSV files are present.
func (reader *Reader) ValidateStructure() []error {
	// Check if the archive can be opened
	allerrs := []error{}
	exists := reader.Adapter.Exists()
	if !exists {
		allerrs = append(allerrs, causes.NewSourceUnreadableError("file does not exist", nil))
		return allerrs
	}
	// Check if these files contain valid headers
	check := func(ent gotransit.Entity) []error {
		fileerrs := []error{}
		efn := ent.Filename()
		err := reader.Adapter.OpenFile(efn, func(in io.Reader) {
			r := csv.NewReader(utfbom.SkipOnly(in))
			row, err := r.Read()
			if err != nil {
				fileerrs = append(fileerrs, err)
				return
			}
			if len(row) == 0 {
				fileerrs = append(fileerrs, errors.New("no data"))
			}
			// Check for column duplicates
			columns := map[string]int{}
			for _, h := range row {
				columns[strings.TrimSpace(h)]++
			}
			for k, v := range columns {
				if v > 1 {
					fileerrs = append(fileerrs, causes.NewFileDuplicateFieldError(efn, k))
				}
			}
			// Ensure we have at least one matching column ID.
			// TODO: Use this to bypass individual entity RequiredFieldErrors?
			// -- maybe have a hierarchy of errors that suppress similar errors?
			ftm := tags.GetStructTagMap(ent)
			for _, field := range ftm {
				if _, ok := columns[field.Csv]; field.Required && !ok {
					fileerrs = append(fileerrs, causes.NewFileRequiredFieldError(efn, field.Csv))
				}
			}
		})
		if err != nil {
			fileerrs = append(fileerrs, causes.NewFileRequiredError(efn))
		}
		return fileerrs
	}
	allerrs = append(allerrs, check(&gotransit.Stop{})...)
	allerrs = append(allerrs, check(&gotransit.Route{})...)
	allerrs = append(allerrs, check(&gotransit.Agency{})...)
	allerrs = append(allerrs, check(&gotransit.Trip{})...)
	allerrs = append(allerrs, check(&gotransit.StopTime{})...)
	cal := gotransit.Calendar{}
	cd := gotransit.CalendarDate{}
	if reader.ContainsFile(cal.Filename()) && reader.ContainsFile(cd.Filename()) {
		allerrs = append(allerrs, check(&cal)...)
		allerrs = append(allerrs, check(&cd)...)
	} else if reader.ContainsFile(cal.Filename()) {
		allerrs = append(allerrs, check(&cal)...)
	} else if reader.ContainsFile(cd.Filename()) {
		allerrs = append(allerrs, check(&cd)...)
	} else {
		allerrs = append(allerrs, causes.NewFileRequiredError(cal.Filename()))
		allerrs = append(allerrs, causes.NewFileRequiredError(cd.Filename()))
	}
	return allerrs
}

// ContainsFile checks if filename is present and contains a readable row.
func (reader *Reader) ContainsFile(filename string) bool {
	// First check if we can read the file
	err := reader.Adapter.OpenFile(filename, func(in io.Reader) {})
	if err != nil {
		return false
	}
	return true
}

// StopTimesByTripID sends StopTimes for selected trips.
func (reader *Reader) StopTimesByTripID(tripIDs ...string) chan []gotransit.StopTime {
	chunks := s2D{}
	grouped := false
	// Get chunks and check if the file is already grouped by ID
	if len(tripIDs) == 0 {
		grouped = true
		counter := map[string]int{}
		last := ""
		// for ent := range reader.StopTimes() {
		reader.Adapter.ReadRows("stop_times.txt", func(row Row) {
			// Only check trip_id
			sid, _ := row.Get("trip_id")
			// If ID transition, have we seen this ID
			if sid != last && grouped == true && last != "" {
				if _, ok := counter[sid]; ok {
					grouped = false
				}
			}
			counter[sid]++
			last = sid
		})
		if grouped {
			keys := []string{}
			for k := range counter {
				keys = append(keys, k)
			}
			chunks = s2D{keys}
		} else {
			chunks = chunkMSI(counter, chunkSize)
		}
	} else {
		chunks = s2D{tripIDs}
	}
	//
	out := make(chan []gotransit.StopTime, bufferSize)
	go func(chunks s2D, grouped bool) {
		for _, chunk := range chunks {
			set := stringsToSet(chunk)
			m := map[string][]gotransit.StopTime{}
			last := ""
			reader.Adapter.ReadRows("stop_times.txt", func(row Row) {
				sid, _ := row.Get("trip_id")
				if _, ok := set[sid]; ok {
					ent := gotransit.StopTime{}
					loadRowStopTime(&ent, row)
					m[sid] = append(m[sid], ent)
				}
				// If we know the file is grouped, send the stoptimes at transition
				if grouped && sid != last && last != "" {
					v := m[last]
					sort.Slice(v, func(i, j int) bool {
						return v[i].StopSequence < v[j].StopSequence
					})
					out <- v
					delete(m, last)
				}
				last = sid
			})
			for _, v := range m {
				sort.Slice(v, func(i, j int) bool {
					return v[i].StopSequence < v[j].StopSequence
				})
				out <- v
			}
		}
		close(out)
	}(chunks, grouped)
	return out
}

// Shapes sends single-geometry LineString Shapes
func (reader *Reader) Shapes() chan gotransit.Shape {
	out := make(chan gotransit.Shape, bufferSize)
	go func() {
		for shapes := range reader.shapesByShapeID() {
			shape := gotransit.NewShapeFromShapes(shapes)
			shape.ShapeID = shapes[0].ShapeID
			out <- shape
		}
		close(out)
	}()
	return out
}

// shapesByShapeID returns a map with grouped Shapes.
func (reader *Reader) shapesByShapeID(shapeIDs ...string) chan []gotransit.Shape {
	chunks := s2D{}
	grouped := false
	// Get chunks and check if the file is already grouped by ID
	if len(shapeIDs) == 0 {
		grouped = true
		counter := map[string]int{}
		last := ""
		reader.Adapter.ReadRows("shapes.txt", func(row Row) {
			// Only check shape_id
			sid, _ := row.Get("shape_id")
			// If ID transition, have we seen this ID
			if sid != last && grouped == true && last != "" {
				if _, ok := counter[sid]; ok {
					grouped = false
				}
			}
			counter[sid]++
			last = sid
		})
		if grouped {
			keys := []string{}
			for k := range counter {
				keys = append(keys, k)
			}
			chunks = s2D{keys}
		} else {
			chunks = chunkMSI(counter, chunkSize)
		}
	} else {
		chunks = s2D{shapeIDs}
	}
	//
	out := make(chan []gotransit.Shape, bufferSize)
	go func(chunks s2D, grouped bool) {
		for _, chunk := range chunks {
			set := stringsToSet(chunk)
			m := map[string][]gotransit.Shape{}
			last := ""
			reader.Adapter.ReadRows("shapes.txt", func(row Row) {
				sid, _ := row.Get("shape_id")
				if _, ok := set[sid]; ok {
					ent := gotransit.Shape{}
					loadRow(&ent, row)
					m[sid] = append(m[sid], ent)
				}
				// If we know the file is grouped, send the shape at transition
				if grouped && sid != last && last != "" {
					v := m[last]
					sort.Slice(v, func(i, j int) bool {
						return v[i].ShapePtSequence < v[j].ShapePtSequence
					})
					out <- v
					delete(m, last)
				}
				last = sid
			})
			for _, v := range m {
				sort.Slice(v, func(i, j int) bool {
					return v[i].ShapePtSequence < v[j].ShapePtSequence
				})
				out <- v
			}
		}
		close(out)
	}(chunks, grouped)
	return out
}

//////////////////////////////
// Entities
//////////////////////////////

// Stops sends Stops.
func (reader *Reader) Stops() (out chan gotransit.Stop) {
	out = make(chan gotransit.Stop, bufferSize)
	go func() {
		ent := gotransit.Stop{}
		reader.Adapter.ReadRows(ent.Filename(), func(row Row) {
			e := gotransit.Stop{}
			loadRow(&e, row)
			e.Geometry = gotransit.NewPoint(e.StopLon, e.StopLat)
			out <- e
		})
		close(out)
	}()
	return out
}

// StopTimes sends StopTimes.
func (reader *Reader) StopTimes() (out chan gotransit.StopTime) {
	out = make(chan gotransit.StopTime, bufferSize)
	go func() {
		ent := gotransit.StopTime{}
		reader.Adapter.ReadRows(ent.Filename(), func(row Row) {
			e := gotransit.StopTime{}
			loadRowStopTime(&e, row) // e.LoadRow(row.Header, row.Row)
			out <- e
		})
		close(out)
	}()
	return out
}

// Agencies sends Agencies.
func (reader *Reader) Agencies() (out chan gotransit.Agency) {
	out = make(chan gotransit.Agency, bufferSize)
	go func() {
		ent := gotransit.Agency{}
		reader.Adapter.ReadRows(ent.Filename(), func(row Row) {
			e := gotransit.Agency{}
			loadRow(&e, row)
			out <- e
		})
		close(out)
	}()
	return out
}

// Calendars sends Calendars.
func (reader *Reader) Calendars() (out chan gotransit.Calendar) {
	out = make(chan gotransit.Calendar, bufferSize)
	go func() {
		ent := gotransit.Calendar{}
		reader.Adapter.ReadRows(ent.Filename(), func(row Row) {
			e := gotransit.Calendar{}
			loadRow(&e, row)
			out <- e
		})
		close(out)
	}()
	return out
}

// CalendarDates sends CalendarDates.
func (reader *Reader) CalendarDates() (out chan gotransit.CalendarDate) {
	out = make(chan gotransit.CalendarDate, bufferSize)
	go func() {
		ent := gotransit.CalendarDate{}
		reader.Adapter.ReadRows(ent.Filename(), func(row Row) {
			e := gotransit.CalendarDate{}
			loadRow(&e, row)
			out <- e
		})
		close(out)
	}()
	return out
}

// FareAttributes sends FareAttributes.
func (reader *Reader) FareAttributes() (out chan gotransit.FareAttribute) {
	out = make(chan gotransit.FareAttribute, bufferSize)
	go func() {
		ent := gotransit.FareAttribute{}
		reader.Adapter.ReadRows(ent.Filename(), func(row Row) {
			e := gotransit.FareAttribute{}
			loadRow(&e, row)
			out <- e
		})
		close(out)
	}()
	return out
}

// FareRules sends FareRules.
func (reader *Reader) FareRules() (out chan gotransit.FareRule) {
	out = make(chan gotransit.FareRule, bufferSize)
	go func() {
		ent := gotransit.FareRule{}
		reader.Adapter.ReadRows(ent.Filename(), func(row Row) {
			e := gotransit.FareRule{}
			loadRow(&e, row)
			out <- e
		})
		close(out)
	}()
	return out
}

// FeedInfos sends FeedInfos.
func (reader *Reader) FeedInfos() (out chan gotransit.FeedInfo) {
	out = make(chan gotransit.FeedInfo, bufferSize)
	go func() {
		ent := gotransit.FeedInfo{}
		reader.Adapter.ReadRows(ent.Filename(), func(row Row) {
			e := gotransit.FeedInfo{}
			loadRow(&e, row)
			out <- e
		})
		close(out)
	}()
	return out
}

// Frequencies sends Frequencies.
func (reader *Reader) Frequencies() (out chan gotransit.Frequency) {
	out = make(chan gotransit.Frequency, bufferSize)
	go func() {
		ent := gotransit.Frequency{}
		reader.Adapter.ReadRows(ent.Filename(), func(row Row) {
			e := gotransit.Frequency{}
			loadRow(&e, row)
			out <- e
		})
		close(out)
	}()
	return out
}

// Routes sends Routes.
func (reader *Reader) Routes() (out chan gotransit.Route) {
	out = make(chan gotransit.Route, bufferSize)
	go func() {
		ent := gotransit.Route{}
		reader.Adapter.ReadRows(ent.Filename(), func(row Row) {
			e := gotransit.Route{}
			loadRow(&e, row)
			out <- e
		})
		close(out)
	}()
	return out
}

// Transfers sends Tranfers.
func (reader *Reader) Transfers() (out chan gotransit.Transfer) {
	out = make(chan gotransit.Transfer, bufferSize)
	go func() {
		ent := gotransit.Transfer{}
		reader.Adapter.ReadRows(ent.Filename(), func(row Row) {
			e := gotransit.Transfer{}
			loadRow(&e, row)
			out <- e
		})
		close(out)
	}()
	return out
}

// Trips sends Trips.
func (reader *Reader) Trips() (out chan gotransit.Trip) {
	out = make(chan gotransit.Trip, bufferSize)
	go func() {
		ent := gotransit.Trip{}
		reader.Adapter.ReadRows(ent.Filename(), func(row Row) {
			e := gotransit.Trip{}
			loadRow(&e, row)
			out <- e
		})
		close(out)
	}()
	return out
}

// chunkMSI takes a string counter and chunks it into groups of size <= chunkSize
func chunkMSI(count map[string]int, chunkSize int) s2D {
	result := s2D{}
	keys := []string{}
	// Sort
	for k := range count {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return count[keys[i]] > count[keys[j]]
	})
	// Chunk
	cur := []string{}
	c := 0
	for _, key := range keys {
		cur = append(cur, key)
		c += count[key]
		if c >= chunkSize {
			result = append(result, cur)
			cur = []string{}
			c = 0
		}
	}
	if len(cur) > 0 {
		result = append(result, cur)
	}
	return result
}

// stringsToSet counts the occurrances of each string, can be used as a Set
func stringsToSet(a []string) map[string]int {
	result := map[string]int{}
	for _, i := range a {
		result[i]++
	}
	return result
}
