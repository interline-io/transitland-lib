package merge

import (
	"fmt"
	"time"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/ext/plus"
)

type daterange struct {
	prefix string
	start  time.Time
	end    time.Time
}

func (dr daterange) check(t time.Time) bool {
	if t.Before(dr.start) || t.After(dr.end) {
		return false
	}
	return true
}

// Check calendar dates, clip if necessary, return error if out of range
func (dr daterange) clipCalendar(v *gotransit.Calendar) error {
	if v.EndDate.Before(dr.start) || v.StartDate.After(dr.end) {
		return fmt.Errorf("out of range")
	}
	if v.StartDate.Before(dr.start) || v.StartDate.Equal(dr.start) {
		v.StartDate = dr.start
	}
	if v.EndDate.After(dr.end) || v.EndDate.Equal(dr.end) {
		v.EndDate = dr.end
	}
	return nil
}

// Filter .
type Filter struct {
	currenturl string
	prefix     string
	hashkeys   *gotransit.EntityMap
	daterange  daterange
}

// NewFilter .
func NewFilter() *Filter {
	return &Filter{
		hashkeys: gotransit.NewEntityMap(),
	}
}

// Filter .
func (ef *Filter) Filter(ent gotransit.Entity, emap *gotransit.EntityMap) error {
	switch v := ent.(type) {
	case *gotransit.Agency:
		// key = []string{v.AgencyID, v.AgencyName, v.AgencyURL}
		v.AgencyID = fmt.Sprintf("%s:%s", ef.prefix, v.AgencyID)
	case *gotransit.Route:
		// key = []string{v.RouteID, v.RouteShortName, v.RouteLongName, strconv.Itoa(v.RouteType), v.RouteColor}
		v.RouteID = fmt.Sprintf("%s:%s", ef.prefix, v.RouteID)
	case *gotransit.Stop:
		// key = []string{v.StopID, v.StopCode, v.StopDesc, v.StopName, fmt.Sprintf("%f", v.Coordinates())}
		v.StopID = fmt.Sprintf("%s:%s", ef.prefix, v.StopID)
	case *gotransit.Trip:
		v.TripID = fmt.Sprintf("%s:%s", ef.prefix, v.TripID)
	case *gotransit.Calendar:
		if err := ef.daterange.clipCalendar(v); err != nil {
			return err
		}
		v.ServiceID = fmt.Sprintf("%s:%s", ef.prefix, v.ServiceID)
	case *gotransit.CalendarDate:
		if !ef.daterange.check(v.Date) {
			return fmt.Errorf("out of range")
		}
	case *gotransit.Shape:
		v.ShapeID = fmt.Sprintf("%s:%s", ef.prefix, v.ShapeID)
	case *gotransit.FareAttribute:
		v.FareID = fmt.Sprintf("%s:%s", ef.prefix, v.FareID)
	case *gotransit.FareRule:
	case *plus.RiderCategory:
	case *plus.FarezoneAttribute:
	default:
	}
	return nil
}

// eid := ent.EntityID()
// efn := ent.Filename()
// h := sha1.New()
// h.Write([]byte(strings.Join(key, "\n")))
// hashkey := fmt.Sprintf("%x", h.Sum(nil))
// if len(key) == 0 {
// 	hashkey = ""
// }
// if eid == "" {
// 	// always new
// 	// fmt.Println("anonymous entity", eid, efn)
// } else if foundhash, seenhash := ef.hashkeys.Get(efn, eid); !seenhash {
// 	// fmt.Println("new entity", eid, efn, hashkey, key)
// 	ef.hashkeys.Set(efn, eid, hashkey)
// } else if foundhash == hashkey {
// 	// fmt.Println("seen entity, same hash", eid, efn, hashkey, foundhash, key)
// } else {
// 	fmt.Println("seen entity, new hash", eid, efn, hashkey, foundhash, key)
// }
