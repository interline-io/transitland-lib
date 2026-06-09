package tlcsv

import (
	"context"
	"io"
	"reflect"
	"sort"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
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
	ent, ok := outInner.Interface().(tt.Entity)
	if !ok {
		return causes.NewSourceUnreadableError("not a valid entity", nil)
	}
	go func() {
		reader.Adapter.ReadRows(ent.Filename(), func(row Row) {
			a := reflect.New(outInnerType)
			e := a.Interface().(tt.Entity)
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
	check := func(ent tt.Entity, rowsRequired bool) []error {
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
			if rowcount == 0 && rowsRequired {
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
	allerrs = append(allerrs, check(&gtfs.Stop{}, false)...)
	allerrs = append(allerrs, check(&gtfs.Route{}, true)...)
	allerrs = append(allerrs, check(&gtfs.Agency{}, true)...)
	allerrs = append(allerrs, check(&gtfs.Trip{}, true)...)
	allerrs = append(allerrs, check(&gtfs.StopTime{}, true)...)
	cal := gtfs.Calendar{}
	cd := gtfs.CalendarDate{}
	calerrs := check(&cal, true)
	cderrs := check(&cd, true)
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
	return err == nil
}

// StopTimesByTripID sends StopTimes for selected trips.
func (reader *Reader) StopTimesByTripID(tripIDs ...string) chan []gtfs.StopTime {
	var chunks s2D
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
			if grouped && sid != last && last != "" {
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
	out := make(chan []gtfs.StopTime, bufferSize)
	go func(chunks s2D, grouped bool) {
		for _, chunk := range chunks {
			set := stringsToSet(chunk)
			m := map[string][]gtfs.StopTime{}
			last := ""
			reader.Adapter.ReadRows("stop_times.txt", func(row Row) {
				sid, _ := row.Get("trip_id")
				if _, ok := set[sid]; ok {
					ent := gtfs.StopTime{}
					loadRow(&ent, row)
					m[sid] = append(m[sid], ent)
				}
				// If we know the file is grouped, send the stoptimes at transition
				if grouped && sid != last && last != "" {
					v := m[last]
					sort.Slice(v, func(i, j int) bool {
						return v[i].StopSequence.Val < v[j].StopSequence.Val
					})
					out <- v
					delete(m, last)
				}
				last = sid
			})
			for _, v := range m {
				sort.Slice(v, func(i, j int) bool {
					return v[i].StopSequence.Val < v[j].StopSequence.Val
				})
				out <- v
			}
		}
		close(out)
	}(chunks, grouped)
	return out
}

// TripsWithStopTimes yields every trip with its StopTimes attached, buffering at
// most chunkSize stop_times at a time (configurable via TL_GTFS_CHUNKSIZE) so memory
// stays bounded regardless of feed size. It satisfies the copier's trip+stop_time
// streaming contract: trips are yielded in stop_times.txt first-appearance order
// (a deterministic order that order-sensitive validators like block-overlap can rely
// on), with trips that have no stop_times trailing in trips.txt order; for a
// duplicate trip_id the first occurrence carries the stop_times and later ones are
// empty; and stop_times whose trip_id is absent from trips.txt are yielded as invalid
// entries (Valid = false) so their reference can still be validated.
//
// Using stop_times order (rather than trips.txt order) is what lets the grouped
// fast path below stream a sorted stop_times.txt in a single pass while producing
// output identical to the chunked path. If ids are given, only those trip_ids are
// yielded.
func (reader *Reader) TripsWithStopTimes(ids ...string) chan gtfs.TripStopTimes {
	out := make(chan gtfs.TripStopTimes, bufferSize)
	go func() {
		defer close(out)
		keep := keepFilter(ids)
		counts, order, grouped := reader.scanTripStopTimes(keep)
		if grouped {
			order = nil
			reader.streamTripStopTimes(out, keep)
		} else {
			reader.chunkTripStopTimes(out, order, counts)
		}
		// Trips with no stop_times never appeared above; emit them last, in trips.txt
		// order (counts is now the set of trip_ids that had stop_times).
		reader.Adapter.ReadRows("trips.txt", func(row Row) {
			tid, _ := row.Get("trip_id")
			if !keep(tid) {
				return
			}
			if _, ok := counts[tid]; ok {
				return
			}
			e := gtfs.Trip{}
			loadRow(&e, row)
			out <- gtfs.TripStopTimes{Valid: true, Trip: e}
		})
	}()
	return out
}

// keepFilter returns a predicate accepting every id when ids is empty, otherwise
// only the listed ones.
func keepFilter(ids []string) func(string) bool {
	set := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		set[id] = struct{}{}
	}
	return func(id string) bool {
		if len(ids) == 0 {
			return true
		}
		_, ok := set[id]
		return ok
	}
}

// scanTripStopTimes reads stop_times.txt once, returning per-trip stop_time counts,
// the order trip_ids first appear (the output order), and whether the file is grouped
// by trip_id (each id's rows contiguous).
func (reader *Reader) scanTripStopTimes(keep func(string) bool) (counts map[string]int, order []string, grouped bool) {
	counts = map[string]int{}
	grouped = true
	last := ""
	reader.Adapter.ReadRows("stop_times.txt", func(row Row) {
		sid, _ := row.Get("trip_id")
		if !keep(sid) {
			return
		}
		_, seen := counts[sid]
		if grouped && sid != last && last != "" && seen {
			grouped = false
		}
		if !seen {
			order = append(order, sid)
		}
		counts[sid]++
		last = sid
	})
	return counts, order, grouped
}

// emitTripChunk reads trips.txt for one chunk's trip_ids and emits them in chunkIDs
// order, each joined with its stop_times (sorted by stop_sequence). A chunk member
// missing from trips.txt is an orphan (Valid = false); duplicate trip rows trail the
// first, empty. Shared by both paths so they produce identical output.
func (reader *Reader) emitTripChunk(out chan<- gtfs.TripStopTimes, chunkIDs []string, stm map[string][]gtfs.StopTime) {
	set := stringsToSet(chunkIDs)
	tripRows := map[string][]gtfs.Trip{}
	reader.Adapter.ReadRows("trips.txt", func(row Row) {
		tid, _ := row.Get("trip_id")
		if _, ok := set[tid]; !ok {
			return
		}
		e := gtfs.Trip{}
		loadRow(&e, row)
		tripRows[tid] = append(tripRows[tid], e)
	})
	for _, tid := range chunkIDs {
		sts := stm[tid]
		sort.Slice(sts, func(i, j int) bool { return sts[i].StopSequence.Val < sts[j].StopSequence.Val })
		rows := tripRows[tid]
		if len(rows) == 0 {
			out <- gtfs.TripStopTimes{StopTimes: sts}
			continue
		}
		out <- gtfs.TripStopTimes{Valid: true, Trip: rows[0], StopTimes: sts}
		for _, dup := range rows[1:] {
			out <- gtfs.TripStopTimes{Valid: true, Trip: dup}
		}
	}
}

// streamTripStopTimes handles a grouped stop_times.txt: a single streaming pass
// gathers each trip's stop_times (only the current chunk is held) and flushes a
// chunk's worth at a time.
func (reader *Reader) streamTripStopTimes(out chan<- gtfs.TripStopTimes, keep func(string) bool) {
	var chunkIDs []string
	stm := map[string][]gtfs.StopTime{}
	count, num := 0, 0
	flush := func() {
		if len(chunkIDs) == 0 {
			return
		}
		num++
		log.For(context.TODO()).Trace().Int("chunk", num).Int("trips", len(chunkIDs)).Msg("tlcsv: processing trip chunk")
		reader.emitTripChunk(out, chunkIDs, stm)
		chunkIDs, stm, count = nil, map[string][]gtfs.StopTime{}, 0
	}
	// A trip is complete once a new trip_id appears (the file is grouped).
	complete := func(tid string) {
		chunkIDs = append(chunkIDs, tid)
		count += len(stm[tid])
		if count >= chunkSize {
			flush()
		}
	}
	last := ""
	reader.Adapter.ReadRows("stop_times.txt", func(row Row) {
		sid, _ := row.Get("trip_id")
		if !keep(sid) {
			return
		}
		if last != "" && sid != last {
			complete(last)
		}
		st := gtfs.StopTime{}
		loadRow(&st, row)
		stm[sid] = append(stm[sid], st)
		last = sid
	})
	if last != "" {
		complete(last)
	}
	flush()
}

// chunkTripStopTimes handles an ungrouped stop_times.txt: it splits the
// first-appearance order into chunks bounded by chunkSize, then re-reads
// stop_times.txt once per chunk to gather that chunk's records.
func (reader *Reader) chunkTripStopTimes(out chan<- gtfs.TripStopTimes, order []string, counts map[string]int) {
	var chunks s2D
	var cur []string
	c := 0
	for _, tid := range order {
		cur = append(cur, tid)
		c += counts[tid]
		if c >= chunkSize {
			chunks = append(chunks, cur)
			cur, c = nil, 0
		}
	}
	if len(cur) > 0 {
		chunks = append(chunks, cur)
	}
	for i, chunk := range chunks {
		log.For(context.TODO()).Trace().
			Int("chunk", i+1).
			Int("chunks", len(chunks)).
			Int("trips", len(chunk)).
			Msg("tlcsv: processing trip chunk")
		set := stringsToSet(chunk)
		stm := map[string][]gtfs.StopTime{}
		reader.Adapter.ReadRows("stop_times.txt", func(row Row) {
			sid, _ := row.Get("trip_id")
			if _, ok := set[sid]; !ok {
				return
			}
			st := gtfs.StopTime{}
			loadRow(&st, row)
			stm[sid] = append(stm[sid], st)
		})
		reader.emitTripChunk(out, chunk, stm)
	}
}

// Shapes sends single-geometry LineString Shapes
func (reader *Reader) Shapes() chan gtfs.Shape {
	return ReadEntities[gtfs.Shape](reader, getFilename(&gtfs.Shape{}))
}

// ShapesByShapeID yields each shape's points grouped into one []gtfs.Shape, in
// shapes.txt order. A grouped file (each shape_id's rows contiguous) streams in a
// single pass; otherwise points are chunked under chunkSize by first-appearance
// order — the same order-preserving scheme as TripsWithStopTimes. With shapeIDs,
// only those shapes are yielded.
func (reader *Reader) ShapesByShapeID(shapeIDs ...string) chan []gtfs.Shape {
	var chunks s2D
	grouped := false
	// Get chunks and check if the file is already grouped by ID
	if len(shapeIDs) == 0 {
		grouped = true
		counter := map[string]int{}
		var order []string // shape_ids in first-appearance order
		last := ""
		reader.Adapter.ReadRows("shapes.txt", func(row Row) {
			sid, _ := row.Get("shape_id")
			_, seen := counter[sid]
			// A shape_id reappearing after a gap means the file isn't grouped by ID.
			if grouped && sid != last && last != "" && seen {
				grouped = false
			}
			if !seen {
				order = append(order, sid)
			}
			counter[sid]++
			last = sid
		})
		if grouped {
			// One streaming pass emits in file order (below); a single chunk suffices.
			chunks = s2D{order}
		} else {
			// Interleaved shape_ids: chunk in first-appearance order so shapes still
			// emit in shapes.txt order, each chunk's points bounded by chunkSize.
			var cur []string
			c := 0
			for _, sid := range order {
				cur = append(cur, sid)
				c += counter[sid]
				if c >= chunkSize {
					chunks = append(chunks, cur)
					cur = nil
					c = 0
				}
			}
			if len(cur) > 0 {
				chunks = append(chunks, cur)
			}
		}
	} else {
		chunks = s2D{shapeIDs}
	}
	//
	out := make(chan []gtfs.Shape, bufferSize)
	go func(chunks s2D, grouped bool) {
		for _, chunk := range chunks {
			set := stringsToSet(chunk)
			m := map[string][]gtfs.Shape{}
			last := ""
			reader.Adapter.ReadRows("shapes.txt", func(row Row) {
				sid, _ := row.Get("shape_id")
				if _, ok := set[sid]; ok {
					ent := gtfs.Shape{}
					loadRow(&ent, row)
					m[sid] = append(m[sid], ent)
				}
				// If we know the file is grouped, send the shape at transition
				if grouped && sid != last && last != "" {
					v := m[last]
					sort.Slice(v, func(i, j int) bool {
						return v[i].ShapePtSequence.Val < v[j].ShapePtSequence.Val
					})
					out <- v
					delete(m, last)
				}
				last = sid
			})
			// Emit remaining shapes in first-appearance (chunk) order: for a grouped
			// file that's just the final shape; otherwise the whole chunk.
			for _, sid := range chunk {
				v, ok := m[sid]
				if !ok {
					continue
				}
				sort.Slice(v, func(i, j int) bool {
					return v[i].ShapePtSequence.Val < v[j].ShapePtSequence.Val
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

func (reader *Reader) Stops() (out chan gtfs.Stop) {
	out = make(chan gtfs.Stop, bufferSize)
	go func() {
		ent := gtfs.Stop{}
		reader.Adapter.ReadRows(ent.Filename(), func(row Row) {
			e := gtfs.Stop{}
			loadRow(&e, row)
			if e.StopLon.Valid && e.StopLat.Valid {
				e.SetCoordinates([2]float64{e.StopLon.Val, e.StopLat.Val})
			}
			out <- e
		})
		close(out)
	}()
	return out
}

func (reader *Reader) StopTimes() (out chan gtfs.StopTime) {
	return ReadEntities[gtfs.StopTime](reader, getFilename(&gtfs.StopTime{}))
}

func (reader *Reader) Agencies() (out chan gtfs.Agency) {
	return ReadEntities[gtfs.Agency](reader, getFilename(&gtfs.Agency{}))
}

func (reader *Reader) Calendars() (out chan gtfs.Calendar) {
	return ReadEntities[gtfs.Calendar](reader, getFilename(&gtfs.Calendar{}))
}

func (reader *Reader) CalendarDates() (out chan gtfs.CalendarDate) {
	return ReadEntities[gtfs.CalendarDate](reader, getFilename(&gtfs.CalendarDate{}))
}

func (reader *Reader) FareAttributes() (out chan gtfs.FareAttribute) {
	return ReadEntities[gtfs.FareAttribute](reader, getFilename(&gtfs.FareAttribute{}))
}

func (reader *Reader) FareRules() (out chan gtfs.FareRule) {
	return ReadEntities[gtfs.FareRule](reader, getFilename(&gtfs.FareRule{}))
}

func (reader *Reader) FeedInfos() (out chan gtfs.FeedInfo) {
	return ReadEntities[gtfs.FeedInfo](reader, getFilename(&gtfs.FeedInfo{}))
}

func (reader *Reader) Frequencies() (out chan gtfs.Frequency) {
	return ReadEntities[gtfs.Frequency](reader, getFilename(&gtfs.Frequency{}))
}

func (reader *Reader) Routes() (out chan gtfs.Route) {
	return ReadEntities[gtfs.Route](reader, getFilename(&gtfs.Route{}))
}

func (reader *Reader) Transfers() (out chan gtfs.Transfer) {
	return ReadEntities[gtfs.Transfer](reader, getFilename(&gtfs.Transfer{}))
}

func (reader *Reader) Trips() (out chan gtfs.Trip) {
	return ReadEntities[gtfs.Trip](reader, getFilename(&gtfs.Trip{}))
}

func (reader *Reader) Levels() (out chan gtfs.Level) {
	return ReadEntities[gtfs.Level](reader, getFilename(&gtfs.Level{}))
}

func (reader *Reader) Pathways() (out chan gtfs.Pathway) {
	return ReadEntities[gtfs.Pathway](reader, getFilename(&gtfs.Pathway{}))
}

func (reader *Reader) Attributions() (out chan gtfs.Attribution) {
	return ReadEntities[gtfs.Attribution](reader, getFilename(&gtfs.Attribution{}))
}

func (reader *Reader) Translations() (out chan gtfs.Translation) {
	return ReadEntities[gtfs.Translation](reader, getFilename(&gtfs.Translation{}))
}

func (reader *Reader) Areas() (out chan gtfs.Area) {
	return ReadEntities[gtfs.Area](reader, getFilename(&gtfs.Area{}))
}

func (reader *Reader) StopAreas() (out chan gtfs.StopArea) {
	return ReadEntities[gtfs.StopArea](reader, getFilename(&gtfs.StopArea{}))
}

func (reader *Reader) FareLegRules() (out chan gtfs.FareLegRule) {
	return ReadEntities[gtfs.FareLegRule](reader, getFilename(&gtfs.FareLegRule{}))
}

func (reader *Reader) FareTransferRules() (out chan gtfs.FareTransferRule) {
	return ReadEntities[gtfs.FareTransferRule](reader, getFilename(&gtfs.FareTransferRule{}))
}

func (reader *Reader) FareProducts() (out chan gtfs.FareProduct) {
	return ReadEntities[gtfs.FareProduct](reader, getFilename(&gtfs.FareProduct{}))
}

func (reader *Reader) FareMedia() (out chan gtfs.FareMedia) {
	return ReadEntities[gtfs.FareMedia](reader, getFilename(&gtfs.FareMedia{}))
}

func (reader *Reader) RiderCategories() (out chan gtfs.RiderCategory) {
	return ReadEntities[gtfs.RiderCategory](reader, getFilename(&gtfs.RiderCategory{}))
}

func (reader *Reader) Timeframes() (out chan gtfs.Timeframe) {
	return ReadEntities[gtfs.Timeframe](reader, getFilename(&gtfs.Timeframe{}))
}

func (reader *Reader) Networks() (out chan gtfs.Network) {
	return ReadEntities[gtfs.Network](reader, getFilename(&gtfs.Network{}))
}

func (reader *Reader) RouteNetworks() (out chan gtfs.RouteNetwork) {
	return ReadEntities[gtfs.RouteNetwork](reader, getFilename(&gtfs.RouteNetwork{}))
}

func (reader *Reader) LocationGroups() (out chan gtfs.LocationGroup) {
	return ReadEntities[gtfs.LocationGroup](reader, getFilename(&gtfs.LocationGroup{}))
}

func (reader *Reader) LocationGroupStops() (out chan gtfs.LocationGroupStop) {
	return ReadEntities[gtfs.LocationGroupStop](reader, getFilename(&gtfs.LocationGroupStop{}))
}

func (reader *Reader) BookingRules() (out chan gtfs.BookingRule) {
	return ReadEntities[gtfs.BookingRule](reader, getFilename(&gtfs.BookingRule{}))
}

func (reader *Reader) Locations() (out chan gtfs.Location) {
	// GTFS-Flex: locations.geojson uses GeoJSON format, not CSV
	// Try to read locations.geojson first
	out = make(chan gtfs.Location, bufferSize)
	go func() {
		defer close(out)

		locs, err := reader.readLocationsGeoJSON("locations.geojson")
		if err != nil {
			// File doesn't exist or error reading - just return empty
			return
		}

		for _, loc := range locs {
			out <- loc
		}
	}()
	return out
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

func getFilename(ent tt.Entity) string {
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
