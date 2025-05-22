package tldb

import (
	"context"
	"errors"
	"reflect"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tt"
)

// Reader reads from a database.
type Reader struct {
	Adapter        Adapter
	PageSize       int
	FeedVersionIDs []int
}

// NewReader returns an initialized Reader based on the provided url string.
func NewReader(dburl string) (*Reader, error) {
	fvids, newurl, err := getFvids(dburl)
	if err != nil {
		return nil, err
	}
	adapter := newAdapter(newurl)
	if adapter == nil {
		return nil, errors.New("no adapter available")
	}
	return &Reader{Adapter: adapter, PageSize: 1_000, FeedVersionIDs: fvids}, nil
}

func (reader *Reader) String() string {
	return "db"
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

// Where returns a select builder with feed_version_id set
func (reader *Reader) Where() sq.SelectBuilder {
	q := reader.Adapter.Sqrl().Select("*")
	if len(reader.FeedVersionIDs) == 1 {
		return q.Where("feed_version_id = ?", reader.FeedVersionIDs[0])
	} else if len(reader.FeedVersionIDs) > 1 {
		return q.Where(sq.Eq{"feed_version_id": reader.FeedVersionIDs})
	}
	return q
}

// ReadEntities provides a generic interface for reading entities.
func (reader *Reader) ReadEntities(c interface{}) error {
	// Seems to work.
	ctx := context.TODO()
	outValue := reflect.ValueOf(c)
	outInnerType := outValue.Type().Elem()
	outInner := reflect.New(outInnerType)
	ent, ok := outInner.Interface().(tt.Entity)
	if !ok {
		return causes.NewSourceUnreadableError("not an entity", nil)
	}
	slice := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(outInner.Interface())), 0, 0)
	// Create a pointer to a slice value and set it to the slice
	x := reflect.New(slice.Type())
	z := x.Elem()
	z.Set(slice)
	//
	qstr, args, err := reader.Where().From(GetTableName(ent)).ToSql()
	if err != nil {
		return err
	}
	if err := reader.Adapter.Select(ctx, x.Interface(), qstr, args...); err != nil {
		return err
	}
	go func() {
		for i := 0; i < z.Len(); i++ {
			p := z.Index(i)
			outValue.Send(p.Elem())
		}
		outValue.Close()
	}()
	return nil
}

// StopTimesByTripID sends StopTimes grouped by TripID.
// Each group is sorted by stop_sequence.
func (reader *Reader) StopTimesByTripID(tripIDs ...string) chan []gtfs.StopTime {
	if len(tripIDs) == 0 {
		q := reader.Adapter.Sqrl().Select("id").Distinct().From("gtfs_trips")
		if len(reader.FeedVersionIDs) == 1 {
			q = q.Where("feed_version_id = ?", reader.FeedVersionIDs[0])
		} else if len(reader.FeedVersionIDs) > 1 {
			q = q.Where(sq.Eq{"feed_version_id": reader.FeedVersionIDs})
		}
		rows, err := q.Query()
		check(err)
		defer rows.Close()
		for rows.Next() {
			tripID := ""
			rows.Scan(&tripID)
			tripIDs = append(tripIDs, tripID)
		}
	}
	out := make(chan []gtfs.StopTime, bufferSize)
	go func() {
		tripChunks := chunkStrings(tripIDs, 1000)
		for _, tripChunk := range tripChunks {
			ents := []gtfs.StopTime{}
			qstr, args, err := reader.Where().From("gtfs_stop_times").Where(sq.Eq{"trip_id": tripChunk}).OrderBy("trip_id", "stop_sequence").ToSql()
			check(err)
			check(reader.Adapter.Select(context.TODO(), &ents, qstr, args...))
			// split by trip
			var cc []gtfs.StopTime
			for _, st := range ents {
				if len(cc) == 0 {
					// ok
				} else if cc[len(cc)-1].TripID != st.TripID {
					out <- cc
					cc = nil
				}
				cc = append(cc, st)
			}
			if len(cc) > 0 {
				out <- cc
			}
		}
		close(out)
	}()
	return out
}

// Shapes sends Shapes grouped by ID.
func (reader *Reader) ShapesByShapeID(ids ...string) chan []gtfs.Shape {
	out := make(chan []gtfs.Shape, bufferSize)
	go func() {
		lastId := 0
		for {
			ents := []service.ShapeLine{}
			qstr, args, err := reader.Where().From("gtfs_shapes").Where(sq.Gt{"id": lastId}).OrderBy("id").Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(context.TODO(), &ents, qstr, args...))
			for _, ent := range ents {
				out <- service.FlattenShape(ent)
				lastId = ent.ID
			}
			if len(ents) < reader.PageSize {
				break
			}
		}
		close(out)
	}()
	return out
}

func (reader *Reader) Shapes() chan gtfs.Shape {
	out := make(chan gtfs.Shape, bufferSize)
	go func() {
		lastId := 0
		for {
			ents := []service.ShapeLine{}
			qstr, args, err := reader.Where().From("gtfs_shapes").Where(sq.Gt{"id": lastId}).OrderBy("id").Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(context.TODO(), &ents, qstr, args...))
			for _, ent := range ents {
				for _, shapeEnt := range service.FlattenShape(ent) {
					out <- shapeEnt
				}
				lastId = ent.ID
			}
			if len(ents) < reader.PageSize {
				break
			}
		}
		close(out)
	}()
	return out
}

func (reader *Reader) StopTimes() chan gtfs.StopTime {
	out := make(chan gtfs.StopTime, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []gtfs.StopTime{}
			qstr, args, err := reader.Where().From("gtfs_stop_times").Offset(uint64(offset)).Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(context.TODO(), &ents, qstr, args...))
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

func (reader *Reader) Stops() chan gtfs.Stop {
	return ReadEntities[gtfs.Stop](reader, GetTableName(&gtfs.Stop{}))
}

func (reader *Reader) Agencies() chan gtfs.Agency {
	return ReadEntities[gtfs.Agency](reader, GetTableName(&gtfs.Agency{}))
}

func (reader *Reader) Calendars() chan gtfs.Calendar {
	return ReadEntities[gtfs.Calendar](reader, GetTableName(&gtfs.Calendar{}))
}

func (reader *Reader) CalendarDates() chan gtfs.CalendarDate {
	return ReadEntities[gtfs.CalendarDate](reader, GetTableName(&gtfs.CalendarDate{}))
}

func (reader *Reader) FareAttributes() chan gtfs.FareAttribute {
	return ReadEntities[gtfs.FareAttribute](reader, GetTableName(&gtfs.FareAttribute{}))
}

func (reader *Reader) FareRules() chan gtfs.FareRule {
	return ReadEntities[gtfs.FareRule](reader, GetTableName(&gtfs.FareRule{}))
}

func (reader *Reader) FeedInfos() chan gtfs.FeedInfo {
	return ReadEntities[gtfs.FeedInfo](reader, GetTableName(&gtfs.FeedInfo{}))
}

func (reader *Reader) Frequencies() chan gtfs.Frequency {
	return ReadEntities[gtfs.Frequency](reader, GetTableName(&gtfs.Frequency{}))
}

func (reader *Reader) Routes() chan gtfs.Route {
	return ReadEntities[gtfs.Route](reader, GetTableName(&gtfs.Route{}))
}

func (reader *Reader) Transfers() chan gtfs.Transfer {
	return ReadEntities[gtfs.Transfer](reader, GetTableName(&gtfs.Transfer{}))
}

func (reader *Reader) Pathways() chan gtfs.Pathway {
	return ReadEntities[gtfs.Pathway](reader, GetTableName(&gtfs.Pathway{}))
}

func (reader *Reader) Levels() chan gtfs.Level {
	return ReadEntities[gtfs.Level](reader, GetTableName(&gtfs.Level{}))
}

func (reader *Reader) Trips() chan gtfs.Trip {
	return ReadEntities[gtfs.Trip](reader, GetTableName(&gtfs.Trip{}))
}

func (reader *Reader) Attributions() chan gtfs.Attribution {
	return ReadEntities[gtfs.Attribution](reader, GetTableName(&gtfs.Attribution{}))
}

func (reader *Reader) Translations() chan gtfs.Translation {
	return ReadEntities[gtfs.Translation](reader, GetTableName(&gtfs.Translation{}))
}

func (reader *Reader) Areas() (out chan gtfs.Area) {
	return ReadEntities[gtfs.Area](reader, GetTableName(&gtfs.Area{}))
}

func (reader *Reader) StopAreas() (out chan gtfs.StopArea) {
	return ReadEntities[gtfs.StopArea](reader, GetTableName(&gtfs.StopArea{}))
}

func (reader *Reader) FareLegRules() (out chan gtfs.FareLegRule) {
	return ReadEntities[gtfs.FareLegRule](reader, GetTableName(&gtfs.FareLegRule{}))
}

func (reader *Reader) FareTransferRules() (out chan gtfs.FareTransferRule) {
	return ReadEntities[gtfs.FareTransferRule](reader, GetTableName(&gtfs.FareTransferRule{}))
}

func (reader *Reader) FareProducts() (out chan gtfs.FareProduct) {
	return ReadEntities[gtfs.FareProduct](reader, GetTableName(&gtfs.FareProduct{}))
}

func (reader *Reader) FareMedia() (out chan gtfs.FareMedia) {
	return ReadEntities[gtfs.FareMedia](reader, GetTableName(&gtfs.FareMedia{}))
}

func (reader *Reader) RiderCategories() (out chan gtfs.RiderCategory) {
	return ReadEntities[gtfs.RiderCategory](reader, GetTableName(&gtfs.RiderCategory{}))
}

func (reader *Reader) Timeframes() (out chan gtfs.Timeframe) {
	return ReadEntities[gtfs.Timeframe](reader, GetTableName(&gtfs.Timeframe{}))
}

func (reader *Reader) Networks() (out chan gtfs.Network) {
	return ReadEntities[gtfs.Network](reader, GetTableName(&gtfs.Network{}))
}

func (reader *Reader) RouteNetworks() (out chan gtfs.RouteNetwork) {
	return ReadEntities[gtfs.RouteNetwork](reader, GetTableName(&gtfs.RouteNetwork{}))
}

func ReadEntities[T tt.EntityWithID](reader *Reader, table string) chan T {
	ctx := context.TODO()
	out := make(chan T, bufferSize)
	go func() {
		lastId := 0
		for {
			var ents []T
			qstr, args, err := reader.Where().From(table).Where(sq.Gt{"id": lastId}).OrderBy("id").Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(ctx, &ents, qstr, args...))
			for _, ent := range ents {
				out <- ent
				lastId = ent.GetID()
			}
			if len(ents) < reader.PageSize {
				break
			}
		}
		close(out)
	}()
	return out
}

func chunkStrings(value []string, csize int) [][]string {
	var output [][]string
	var cur []string
	for _, s := range value {
		cur = append(cur, s)
		if len(cur) >= csize {
			output = append(output, cur)
			cur = nil
		}
	}
	if len(cur) > 0 {
		output = append(output, cur)
	}
	return output
}
