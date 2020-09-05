package tldb

import (
	"reflect"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

func check(err error) {
	if err != nil {
		log.Debug("Error: %s", err)
	}
}

// Reader reads from a database.
type Reader struct {
	Adapter        Adapter
	PageSize       int
	FeedVersionIDs []int
}

// NewReader returns an initialized Reader based on the provided url string.
func NewReader(dburl string) (*Reader, error) {
	return &Reader{Adapter: newAdapter(dburl), PageSize: 1000}, nil
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
	outValue := reflect.ValueOf(c)
	outInnerType := outValue.Type().Elem()
	outInner := reflect.New(outInnerType)
	ent, ok := outInner.Interface().(tl.Entity)
	if !ok {
		return causes.NewSourceUnreadableError("not an entity", nil)
	}
	slice := reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(outInner.Interface())), 0, 0)
	// Create a pointer to a slice value and set it to the slice
	x := reflect.New(slice.Type())
	z := x.Elem()
	z.Set(slice)
	//
	qstr, args, err := reader.Where().From(getTableName(ent)).ToSql()
	if err != nil {
		return err
	}
	if err := reader.Adapter.Select(x.Interface(), qstr, args...); err != nil {
		check(err)
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
func (reader *Reader) StopTimesByTripID(tripIDs ...string) chan []tl.StopTime {
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
	out := make(chan []tl.StopTime, bufferSize)
	go func() {
		for _, tripID := range tripIDs {
			ents := []tl.StopTime{}
			qstr, args, err := reader.Where().From("gtfs_stop_times").Where("trip_id = ?", tripID).OrderBy("stop_sequence").ToSql()
			check(err)
			check(reader.Adapter.Select(&ents, qstr, args...))
			if len(ents) > 0 {
				out <- ents
			}
		}
		close(out)
	}()
	return out
}

// Stops sends Stops.
func (reader *Reader) Stops() chan tl.Stop {
	out := make(chan tl.Stop, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []tl.Stop{}
			qstr, args, err := reader.Where().From("gtfs_stops").OrderBy("id").Offset(uint64(offset)).Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(&ents, qstr, args...))
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
func (reader *Reader) StopTimes() chan tl.StopTime {
	out := make(chan tl.StopTime, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []tl.StopTime{}
			qstr, args, err := reader.Where().From("gtfs_stop_times").OrderBy("id").Offset(uint64(offset)).Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(&ents, qstr, args...))
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
func (reader *Reader) Agencies() chan tl.Agency {
	out := make(chan tl.Agency, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []tl.Agency{}
			qstr, args, err := reader.Where().From("gtfs_agencies").OrderBy("id").Offset(uint64(offset)).Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(&ents, qstr, args...))
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
func (reader *Reader) Calendars() chan tl.Calendar {
	out := make(chan tl.Calendar, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []tl.Calendar{}
			qstr, args, err := reader.Where().From("gtfs_calendars").OrderBy("id").Offset(uint64(offset)).Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(&ents, qstr, args...))
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
func (reader *Reader) CalendarDates() chan tl.CalendarDate {
	out := make(chan tl.CalendarDate, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []tl.CalendarDate{}
			qstr, args, err := reader.Where().From("gtfs_calendar_dates").OrderBy("id").Offset(uint64(offset)).Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(&ents, qstr, args...))
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
func (reader *Reader) FareAttributes() chan tl.FareAttribute {
	out := make(chan tl.FareAttribute, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []tl.FareAttribute{}
			qstr, args, err := reader.Where().From("gtfs_fare_attributes").OrderBy("id").Offset(uint64(offset)).Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(&ents, qstr, args...))
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
func (reader *Reader) FareRules() chan tl.FareRule {
	out := make(chan tl.FareRule, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []tl.FareRule{}
			qstr, args, err := reader.Where().From("gtfs_fare_rules").OrderBy("id").Offset(uint64(offset)).Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(&ents, qstr, args...))
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
func (reader *Reader) FeedInfos() chan tl.FeedInfo {
	out := make(chan tl.FeedInfo, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []tl.FeedInfo{}
			qstr, args, err := reader.Where().From("gtfs_feed_infos").OrderBy("id").Offset(uint64(offset)).Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(&ents, qstr, args...))
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
func (reader *Reader) Frequencies() chan tl.Frequency {
	out := make(chan tl.Frequency, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []tl.Frequency{}
			qstr, args, err := reader.Where().From("gtfs_frequencies").OrderBy("id").Offset(uint64(offset)).Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(&ents, qstr, args...))
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
func (reader *Reader) Routes() chan tl.Route {
	out := make(chan tl.Route, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []tl.Route{}
			qstr, args, err := reader.Where().From("gtfs_routes").OrderBy("id").Offset(uint64(offset)).Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(&ents, qstr, args...))
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
func (reader *Reader) Shapes() chan tl.Shape {
	out := make(chan tl.Shape, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []tl.Shape{}
			qstr, args, err := reader.Where().From("gtfs_shapes").OrderBy("id").Offset(uint64(offset)).Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(&ents, qstr, args...))
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
func (reader *Reader) Transfers() chan tl.Transfer {
	out := make(chan tl.Transfer, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []tl.Transfer{}
			qstr, args, err := reader.Where().From("gtfs_transfers").OrderBy("id").Offset(uint64(offset)).Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(&ents, qstr, args...))
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

// Pathways sends Pathways.
func (reader *Reader) Pathways() chan tl.Pathway {
	out := make(chan tl.Pathway, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []tl.Pathway{}
			qstr, args, err := reader.Where().From("gtfs_pathways").OrderBy("id").Offset(uint64(offset)).Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(&ents, qstr, args...))
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

// Levels sends Levels.
func (reader *Reader) Levels() chan tl.Level {
	out := make(chan tl.Level, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []tl.Level{}
			qstr, args, err := reader.Where().From("gtfs_levels").OrderBy("id").Offset(uint64(offset)).Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(&ents, qstr, args...))
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
func (reader *Reader) Trips() chan tl.Trip {
	out := make(chan tl.Trip, bufferSize)
	go func() {
		offset := 0
		for {
			ents := []tl.Trip{}
			qstr, args, err := reader.Where().From("gtfs_trips").OrderBy("id").Offset(uint64(offset)).Limit(uint64(reader.PageSize)).ToSql()
			check(err)
			check(reader.Adapter.Select(&ents, qstr, args...))
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
