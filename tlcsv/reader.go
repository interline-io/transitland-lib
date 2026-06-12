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
			// A file with only a header and no data rows is treated as empty.
			// Required files report FileRequiredError; optional files (e.g.
			// stops.txt in a flex feed) are skipped without error. We return
			// here because the column check below cannot run on a file that has
			// no rows: the CSV header is never surfaced to this callback.
			if rowcount == 0 {
				if rowsRequired {
					fileerrs = append(fileerrs, causes.NewFileRequiredError(efn))
				}
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
	// stops.txt is conditionally required. It may be omitted entirely when the
	// feed provides location-based service via locations.geojson or
	// location_groups.txt (GTFS-Flex). When the file is present it is always
	// validated (an empty, header-only stops.txt is allowed).
	stopsAlternativePresent := reader.ContainsFile((&gtfs.Location{}).Filename()) ||
		reader.ContainsFile((&gtfs.LocationGroup{}).Filename())
	if reader.ContainsFile((&gtfs.Stop{}).Filename()) || !stopsAlternativePresent {
		allerrs = append(allerrs, check(&gtfs.Stop{}, false)...)
	}
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
	out := make(chan []gtfs.StopTime, groupBufferSize)
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
	out := make(chan gtfs.TripStopTimes, groupBufferSize)
	go func() {
		defer close(out)
		filter := make(map[string]struct{}, len(ids))
		for _, id := range ids {
			filter[id] = struct{}{}
		}
		keep := func(id string) bool {
			if len(ids) == 0 {
				return true
			}
			_, ok := filter[id]
			return ok
		}
		counts, order, grouped := reader.scanTripStopTimes(keep)
		if grouped {
			// Sorted fast path: one streaming pass, chunked by trip count (only one
			// trip's stop_times is held at a time, so trips.txt drives the chunk size).
			reader.streamTripStopTimes(out, keep, order)
		} else {
			// Unsorted fallback: re-read stop_times.txt once per stop_time-bounded chunk.
			reader.chunkTripStopTimes(out, order, counts)
		}
		order = nil
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
			// Clone out of the csv reader's per-record string so this long-lived key
			// retains only the trip_id, not the whole pinned stop_times row.
			sid = strings.Clone(sid)
			order = append(order, sid)
		}
		counts[sid]++
		last = sid
	})
	return counts, order, grouped
}

// chunkByCount splits trip_ids (in order) into chunks whose stop_time counts stay
// under chunkSize, bounding how many records a chunk holds at once.
func chunkByCount(order []string, counts map[string]int) s2D {
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
	return chunks
}

// chunkByTrips splits trip_ids (in order) into chunks of at most n trips. The sorted
// path uses this because its live stop_time buffer is already one trip, so the chunk
// size only governs how many trip records are held while reading trips.txt — far cheaper
// than the stop_time budget chunkByCount enforces, so n can be large and the trips.txt
// passes few.
func chunkByTrips(order []string, n int) s2D {
	if n < 1 {
		n = 1
	}
	var chunks s2D
	for i := 0; i < len(order); i += n {
		end := i + n
		if end > len(order) {
			end = len(order)
		}
		chunks = append(chunks, order[i:end])
	}
	return chunks
}

// readTripsForIDs reads trips.txt and returns the rows for the given trip_ids, grouped
// by id in file order so duplicate rows are preserved.
func (reader *Reader) readTripsForIDs(ids []string) map[string][]gtfs.Trip {
	set := stringsToSet(ids)
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
	return tripRows
}

// emitJoinedTrip emits one trip joined with its stop_times (sorted by stop_sequence).
// rows are the trips.txt rows for the id: none means an orphan (Valid = false); any
// extras are duplicate trip rows, emitted after the first with empty StopTimes. When
// capped is set the stop_times were truncated at chunkSize, so the emitted entity
// carries an EntityLimitError.
func emitJoinedTrip(out chan<- gtfs.TripStopTimes, rows []gtfs.Trip, sts []gtfs.StopTime, capped bool) {
	sort.Slice(sts, func(i, j int) bool { return sts[i].StopSequence.Val < sts[j].StopSequence.Val })
	if len(rows) == 0 {
		if capped && len(sts) > 0 {
			sts[0].AddError(causes.NewEntityLimitError(sts[0].TripID.Val, "stop_times", chunkSize))
		}
		out <- gtfs.TripStopTimes{StopTimes: sts}
		return
	}
	trip := rows[0]
	if capped {
		trip.AddError(causes.NewEntityLimitError(trip.EntityID(), "stop_times", chunkSize))
	}
	out <- gtfs.TripStopTimes{Valid: true, Trip: trip, StopTimes: sts}
	for _, dup := range rows[1:] {
		out <- gtfs.TripStopTimes{Valid: true, Trip: dup}
	}
}

// streamTripStopTimes handles a grouped stop_times.txt: one streaming pass completes
// trips in order, holding only the current trip's stop_times (capped at chunkSize). A
// chunk's trips are read from trips.txt when its first trip is reached, so peak memory
// is at most chunkSize stop_times plus one chunk's trips. Because stop_time memory is
// already bounded to one trip, chunks are sized by trip count (chunkSize, counted in
// trips here), keeping trips.txt passes proportional to trips rather than the much
// larger stop_time count.
func (reader *Reader) streamTripStopTimes(out chan<- gtfs.TripStopTimes, keep func(string) bool, order []string) {
	chunks := chunkByTrips(order, chunkSize)
	ci, pi := 0, 0 // current chunk index, position within it
	var chunkTrips map[string][]gtfs.Trip
	emit := func(tid string, sts []gtfs.StopTime, capped bool) {
		if pi == 0 {
			log.For(context.TODO()).Trace().Int("chunk", ci+1).Int("chunks", len(chunks)).Int("trips", len(chunks[ci])).Msg("tlcsv: processing trip chunk")
			chunkTrips = reader.readTripsForIDs(chunks[ci])
		}
		emitJoinedTrip(out, chunkTrips[tid], sts, capped)
		if pi++; pi == len(chunks[ci]) {
			ci, pi, chunkTrips = ci+1, 0, nil
		}
	}
	var sts []gtfs.StopTime
	last := ""
	capped := false
	reader.Adapter.ReadRows("stop_times.txt", func(row Row) {
		sid, _ := row.Get("trip_id")
		if !keep(sid) {
			return
		}
		if last != "" && sid != last {
			emit(last, sts, capped)
			sts = nil
			capped = false
		}
		last = sid
		// Cap one trip's stop_times at chunkSize so a degenerate trip can't grow this
		// buffer without bound; the trip is still emitted, flagged via emitJoinedTrip.
		if len(sts) >= chunkSize {
			capped = true
			return
		}
		st := gtfs.StopTime{}
		loadRow(&st, row)
		sts = append(sts, st)
	})
	if last != "" {
		emit(last, sts, capped)
	}
}

// chunkTripStopTimes handles an ungrouped stop_times.txt: each chunk re-reads
// stop_times.txt to gather its scattered records, then emits. Chunks are sized by
// stop_time count (chunkByCount) because a chunk buffers all of its stop_times at once.
func (reader *Reader) chunkTripStopTimes(out chan<- gtfs.TripStopTimes, order []string, counts map[string]int) {
	chunks := chunkByCount(order, counts)
	for i, chunk := range chunks {
		log.For(context.TODO()).Trace().
			Int("chunk", i+1).
			Int("chunks", len(chunks)).
			Int("trips", len(chunk)).
			Msg("tlcsv: processing trip chunk")
		set := stringsToSet(chunk)
		stm := map[string][]gtfs.StopTime{}
		capped := map[string]bool{}
		reader.Adapter.ReadRows("stop_times.txt", func(row Row) {
			sid, _ := row.Get("trip_id")
			if _, ok := set[sid]; !ok {
				return
			}
			// Cap each trip's stop_times at chunkSize (see streamTripStopTimes).
			if len(stm[sid]) >= chunkSize {
				capped[sid] = true
				return
			}
			st := gtfs.StopTime{}
			loadRow(&st, row)
			stm[sid] = append(stm[sid], st)
		})
		tripRows := reader.readTripsForIDs(chunk)
		for _, tid := range chunk {
			emitJoinedTrip(out, tripRows[tid], stm[tid], capped[tid])
		}
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
	out := make(chan []gtfs.Shape, groupBufferSize)
	go func(chunks s2D, grouped bool) {
		for _, chunk := range chunks {
			set := stringsToSet(chunk)
			m := map[string][]gtfs.Shape{}
			capped := map[string]bool{}
			// emitShape sorts a shape's points by sequence, flags it if its points were
			// capped at chunkSize, sends it, and drops it from the buffer.
			emitShape := func(sid string) {
				v := m[sid]
				sort.Slice(v, func(i, j int) bool {
					return v[i].ShapePtSequence.Val < v[j].ShapePtSequence.Val
				})
				if capped[sid] && len(v) > 0 {
					v[0].AddError(causes.NewEntityLimitError(sid, "shapes", chunkSize))
				}
				out <- v
				delete(m, sid)
			}
			last := ""
			reader.Adapter.ReadRows("shapes.txt", func(row Row) {
				sid, _ := row.Get("shape_id")
				if _, ok := set[sid]; ok {
					// Cap one shape's points at chunkSize so a degenerate shape can't grow
					// this buffer without bound; it is still emitted, flagged.
					if len(m[sid]) >= chunkSize {
						capped[sid] = true
					} else {
						ent := gtfs.Shape{}
						loadRow(&ent, row)
						m[sid] = append(m[sid], ent)
					}
				}
				// If we know the file is grouped, send the shape at transition
				if grouped && sid != last && last != "" {
					emitShape(last)
				}
				last = sid
			})
			// Emit remaining shapes in first-appearance (chunk) order: for a grouped
			// file that's just the final shape; otherwise the whole chunk.
			for _, sid := range chunk {
				if _, ok := m[sid]; !ok {
					continue
				}
				emitShape(sid)
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
