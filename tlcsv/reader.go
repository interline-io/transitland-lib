package tlcsv

import (
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

// s2D is two dimensional string slice
type s2D = [][]string

// Reader reads GTFS entities from CSV files.
type Reader struct {
	Adapter
}

func NewReaderFromAdapter(a Adapter) (*Reader, error) {
	return &Reader{Adapter: a}, nil
}

// NewReader returns an initialized CSV Reader.
func NewReader(path string) (*Reader, error) {
	a, err := NewAdapter(path)
	if err != nil {
		return nil, err
	}
	return &Reader{Adapter: a}, nil
}

func (reader *Reader) String() string {
	return reader.Adapter.String()
}

// ReadEntities provides a generic interface for reading entities.
func (reader *Reader) ReadEntities(c interface{}) error {
	// Magic
	outValue := reflect.ValueOf(c)
	outInnerType := outValue.Type().Elem()
	outInner := reflect.New(outInnerType)
	ent, ok := outInner.Interface().(tl.Entity)
	if !ok {
		return causes.NewSourceUnreadableError("not a valid entity", nil)
	}
	go func() {
		reader.Adapter.ReadRows(ent.Filename(), func(row Row) {
			a := reflect.New(outInnerType)
			e := a.Interface().(tl.Entity)
			loadRow(e, row)
			outValue.Send(a.Elem())
		})
		outValue.Close()
	}()
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
	// TODO: An error in the header should also stop a file from being opened for further CSV reading.
	check := func(ent tl.Entity) []error {
		fileerrs := []error{}
		efn := ent.Filename()
		err := reader.Adapter.OpenFile(efn, func(in io.Reader) {
			rowcount := 0
			rowheader := []string{}
			readerr := ReadRows(in, func(row Row) {
				if len(rowheader) == 0 {
					rowheader = row.Header
				}
				rowcount++
			})
			// If the file is unreadable or has no rows then return
			if readerr != nil {
				fileerrs = append(fileerrs, causes.NewFileUnreadableError(efn, readerr))
				return
			}
			if rowcount == 0 {
				fileerrs = append(fileerrs, causes.NewFileRequiredError(efn))
				return
			}
			// Check columns
			columns := map[string]int{}
			for _, h := range rowheader {
				columns[strings.TrimSpace(h)]++
			}
			// Ensure we have at least one matching column ID.
			found := []string{}
			missing := []string{}
			for _, field := range MapperCache.GetStructTagMap(ent) {
				if _, ok := columns[field.Name]; ok {
					found = append(found, field.Name)
				} else if field.Required {
					missing = append(missing, field.Name)
				}
			}
			if len(found) == 0 {
				fileerrs = append(fileerrs, causes.NewFileRequiredError(efn))
				return
			}
			if len(missing) > 0 {
				for _, field := range missing {
					fileerrs = append(fileerrs, causes.NewFileRequiredFieldError(efn, field))
				}
			}
			// Check for column duplicates
			for k, v := range columns {
				if v > 1 {
					fileerrs = append(fileerrs, causes.NewFileDuplicateFieldError(efn, k))
				}
			}
		})
		if err != nil {
			fileerrs = append(fileerrs, causes.NewFileRequiredError(efn))
		}
		return fileerrs
	}
	allerrs = append(allerrs, check(&tl.Stop{})...)
	allerrs = append(allerrs, check(&tl.Route{})...)
	allerrs = append(allerrs, check(&tl.Agency{})...)
	allerrs = append(allerrs, check(&tl.Trip{})...)
	allerrs = append(allerrs, check(&tl.StopTime{})...)
	cal := tl.Calendar{}
	cd := tl.CalendarDate{}
	calerrs := check(&cal)
	cderrs := check(&cd)
	if reader.ContainsFile(cal.Filename()) && reader.ContainsFile(cd.Filename()) {
		if len(calerrs) > 0 && len(cderrs) > 0 {
			allerrs = append(allerrs, calerrs...)
			allerrs = append(allerrs, cderrs...)
		}
	} else if reader.ContainsFile(cal.Filename()) {
		allerrs = append(allerrs, calerrs...)
	} else if reader.ContainsFile(cd.Filename()) {
		allerrs = append(allerrs, cderrs...)
	} else {
		allerrs = append(allerrs, calerrs...)
		allerrs = append(allerrs, cderrs...)
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
func (reader *Reader) StopTimesByTripID(tripIDs ...string) chan []tl.StopTime {
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
	out := make(chan []tl.StopTime, bufferSize)
	go func(chunks s2D, grouped bool) {
		for _, chunk := range chunks {
			set := stringsToSet(chunk)
			m := map[string][]tl.StopTime{}
			last := ""
			reader.Adapter.ReadRows("stop_times.txt", func(row Row) {
				sid, _ := row.Get("trip_id")
				if _, ok := set[sid]; ok {
					ent := tl.StopTime{}
					loadRow(&ent, row)
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
func (reader *Reader) Shapes() chan tl.Shape {
	out := make(chan tl.Shape, bufferSize)
	go func() {
		for shapes := range reader.shapesByShapeID() {
			shape := tl.NewShapeFromShapes(shapes)
			shape.ShapeID = shapes[0].ShapeID
			out <- shape
		}
		close(out)
	}()
	return out
}

// shapesByShapeID returns a map with grouped Shapes.
func (reader *Reader) shapesByShapeID(shapeIDs ...string) chan []tl.Shape {
	var chunks s2D
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
	out := make(chan []tl.Shape, bufferSize)
	go func(chunks s2D, grouped bool) {
		for _, chunk := range chunks {
			set := stringsToSet(chunk)
			m := map[string][]tl.Shape{}
			last := ""
			reader.Adapter.ReadRows("shapes.txt", func(row Row) {
				sid, _ := row.Get("shape_id")
				if _, ok := set[sid]; ok {
					ent := tl.Shape{}
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

func (reader *Reader) Stops() (out chan tl.Stop) {
	out = make(chan tl.Stop, bufferSize)
	go func() {
		ent := tl.Stop{}
		reader.Adapter.ReadRows(ent.Filename(), func(row Row) {
			e := tl.Stop{}
			loadRow(&e, row)
			e.SetCoordinates([2]float64{e.StopLon, e.StopLat})
			out <- e
		})
		close(out)
	}()
	return out
}

func (reader *Reader) StopTimes() (out chan tl.StopTime) {
	return ReadEntities[tl.StopTime](reader, getFilename(&tl.StopTime{}))
}

func (reader *Reader) Agencies() (out chan tl.Agency) {
	return ReadEntities[tl.Agency](reader, getFilename(&tl.Agency{}))
}

func (reader *Reader) Calendars() (out chan tl.Calendar) {
	return ReadEntities[tl.Calendar](reader, getFilename(&tl.Calendar{}))
}

func (reader *Reader) CalendarDates() (out chan tl.CalendarDate) {
	return ReadEntities[tl.CalendarDate](reader, getFilename(&tl.CalendarDate{}))
}

func (reader *Reader) FareAttributes() (out chan tl.FareAttribute) {
	return ReadEntities[tl.FareAttribute](reader, getFilename(&tl.FareAttribute{}))
}

func (reader *Reader) FareRules() (out chan tl.FareRule) {
	return ReadEntities[tl.FareRule](reader, getFilename(&tl.FareRule{}))
}

func (reader *Reader) FeedInfos() (out chan tl.FeedInfo) {
	return ReadEntities[tl.FeedInfo](reader, getFilename(&tl.FeedInfo{}))
}

func (reader *Reader) Frequencies() (out chan tl.Frequency) {
	return ReadEntities[tl.Frequency](reader, getFilename(&tl.Frequency{}))
}

func (reader *Reader) Routes() (out chan tl.Route) {
	return ReadEntities[tl.Route](reader, getFilename(&tl.Route{}))
}

func (reader *Reader) Transfers() (out chan tl.Transfer) {
	return ReadEntities[tl.Transfer](reader, getFilename(&tl.Transfer{}))
}

func (reader *Reader) Trips() (out chan tl.Trip) {
	return ReadEntities[tl.Trip](reader, getFilename(&tl.Trip{}))
}

func (reader *Reader) Levels() (out chan tl.Level) {
	return ReadEntities[tl.Level](reader, getFilename(&tl.Level{}))
}

func (reader *Reader) Pathways() (out chan tl.Pathway) {
	return ReadEntities[tl.Pathway](reader, getFilename(&tl.Pathway{}))
}

func (reader *Reader) Attributions() (out chan tl.Attribution) {
	return ReadEntities[tl.Attribution](reader, getFilename(&tl.Attribution{}))
}

func (reader *Reader) Translations() (out chan tl.Translation) {
	return ReadEntities[tl.Translation](reader, getFilename(&tl.Translation{}))
}

func (reader *Reader) Areas() (out chan tl.Area) {
	return ReadEntities[tl.Area](reader, getFilename(&tl.Area{}))
}

func (reader *Reader) StopAreas() (out chan tl.StopArea) {
	return ReadEntities[tl.StopArea](reader, getFilename(&tl.StopArea{}))
}

func (reader *Reader) FareLegRules() (out chan tl.FareLegRule) {
	return ReadEntities[tl.FareLegRule](reader, getFilename(&tl.FareLegRule{}))
}

func (reader *Reader) FareTransferRules() (out chan tl.FareTransferRule) {
	return ReadEntities[tl.FareTransferRule](reader, getFilename(&tl.FareTransferRule{}))
}

func (reader *Reader) FareProducts() (out chan tl.FareProduct) {
	return ReadEntities[tl.FareProduct](reader, getFilename(&tl.FareProduct{}))
}

func (reader *Reader) FareMedia() (out chan tl.FareMedia) {
	return ReadEntities[tl.FareMedia](reader, getFilename(&tl.FareMedia{}))
}

func (reader *Reader) RiderCategories() (out chan tl.RiderCategory) {
	return ReadEntities[tl.RiderCategory](reader, getFilename(&tl.RiderCategory{}))
}

func ReadEntities[T any](reader *Reader, efn string) chan T {
	eout := make(chan T, bufferSize)
	go func(fn string, c chan T) {
		reader.Adapter.ReadRows(fn, func(row Row) {
			var e T
			loadRow(&e, row)
			c <- e
		})
		close(c)
	}(efn, eout)
	return eout
}

func getFilename(ent tl.Entity) string {
	return ent.Filename()
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
