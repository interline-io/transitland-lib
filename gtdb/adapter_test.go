package gtdb

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/copier"
	"github.com/interline-io/gotransit/gtcsv"
)

func filldb(writer *Writer) error {
	r1, err := gtcsv.NewReader("../testdata/example")
	if err != nil {
		return err
	}
	r1.Open()
	defer r1.Close()
	if _, err := writer.CreateFeedVersion(r1); err != nil {
		return err
	}
	cp := copier.NewCopier(r1, writer)
	cp.Copy()
	return nil
}

func testAdapter(t *testing.T, adapter Adapter) {
	// need a feedversion...
	var err error
	var m minEnts
	t.Run("Insert", func(t *testing.T) {
		var err error
		// minEntities uses Insert
		m, err = minEntities(adapter)
		if err != nil {
			fmt.Println("ERR:", err)
			t.Error(err)
			t.FailNow()
		}
	})
	// fmt.Printf("%#v\n", m)
	if err != nil {
		fmt.Println("ERR:", err)
		t.Error(err)
		t.FailNow()
	}
	t.Run("Find", func(t *testing.T) {
		ent := gotransit.Trip{}
		ent.ID = m.TripID
		if err := adapter.Find(&ent); err != nil {
			t.Error(err)
		}
		if ent.ID != m.TripID {
			t.Errorf("got %d expected %d", ent.ID, m.TripID)
		}
	})
	t.Run("Select", func(t *testing.T) {
		ents := []gotransit.Stop{}
		if err := adapter.Select(&ents, "SELECT * FROM gtfs_stops WHERE id IN (?,?) AND feed_version_id = ? ORDER BY id ASC", m.StopID1, m.StopID2, m.FeedVersionID); err != nil {
			t.Error(err)
		}
		if len(ents) == 0 {
			t.Errorf("got no results")
		} else if len(ents) != 2 {
			t.Errorf("got %d expected %d", len(ents), 2)
		} else {
			ent := ents[0]
			if ent.ID != m.StopID1 {
				t.Errorf("got %d expected %d", ent.ID, m.StopID1)
			}
			ent2 := ents[1]
			if ent2.ID != m.StopID2 {
				t.Errorf("got %d expected %d", ent.ID, m.StopID2)
			}
		}
	})
	t.Run("BatchInsert", func(t *testing.T) {
		st1 := gotransit.StopTime{}
		st1.FeedVersionID = m.FeedVersionID
		st1.StopID = strconv.Itoa(m.StopID1)
		st1.TripID = strconv.Itoa(m.TripID)
		st1.StopSequence = 1
		st1.ArrivalTime = 0
		st1.DepartureTime = 1
		st2 := gotransit.StopTime{}
		st2.FeedVersionID = m.FeedVersionID
		st2.StopID = strconv.Itoa(m.StopID2)
		st2.TripID = strconv.Itoa(m.TripID)
		st2.StopSequence = 2
		st2.ArrivalTime = 2
		st2.DepartureTime = 3
		sts := []gotransit.Entity{&st1, &st2}
		if err := adapter.BatchInsert(sts); err != nil {
			t.Error(err)
		}
		sts2 := []gotransit.StopTime{}
		if err := adapter.Select(&sts2, "SELECT * FROM gtfs_stop_times WHERE feed_version_id = ? ORDER BY id ASC", m.FeedVersionID); err != nil {
			t.Error(err)
		}
		if len(sts2) == 0 {
			t.Errorf("got no results")
		} else if len(sts2) != 2 {
			t.Errorf("got %d expected %d", len(sts2), 2)
		} else {
			got1 := sts2[0]
			if v := st1.StopID; got1.StopID != v {
				t.Errorf("got '%s' expected '%s'", got1.StopID, v)
			}
			if v := st1.TripID; got1.TripID != v {
				t.Errorf("got '%s' expected '%s'", got1.TripID, v)
			}
			if got1.StopSequence != st1.StopSequence {
				t.Errorf("got '%d' expected '%d'", got1.StopSequence, st1.StopSequence)
			}
		}
	})
}

type minEnts struct {
	FeedVersionID int
	AgencyID      int
	RouteID       int
	ServiceID     int
	TripID        int
	StopID1       int
	StopID2       int
}

// minEntities creates a minimal number of basic entities,
// with only enough detail to satisfy database constraints.
// This function uses adapter.Insert.
func minEntities(adapter Adapter) (minEnts, error) {
	var err error
	m := minEnts{}
	fv := gotransit.FeedVersion{}
	m.FeedVersionID, err = adapter.Insert(&fv)
	if err != nil {
		return m, err
	}
	ent0 := gotransit.Agency{}
	ent0.AgencyID = "ok"
	ent0.FeedVersionID = m.FeedVersionID
	m.AgencyID, err = adapter.Insert(&ent0)
	if err != nil {
		return m, err
	}
	ent4 := gotransit.Route{}
	ent4.RouteID = "ok"
	ent4.AgencyID = strconv.Itoa(m.AgencyID)
	ent4.FeedVersionID = m.FeedVersionID
	m.RouteID, err = adapter.Insert(&ent4)
	if err != nil {
		return m, err
	}
	cal := gotransit.Calendar{}
	cal.StartDate = time.Now()
	cal.EndDate = time.Now()
	cal.ServiceID = "ok"
	cal.FeedVersionID = m.FeedVersionID
	m.ServiceID, err = adapter.Insert(&cal)
	if err != nil {
		return m, err
	}
	ent1 := gotransit.Trip{}
	ent1.TripID = "ok"
	ent1.RouteID = strconv.Itoa(m.RouteID)
	ent1.ServiceID = strconv.Itoa(m.ServiceID)
	ent1.FeedVersionID = m.FeedVersionID
	m.TripID, err = adapter.Insert(&ent1)
	if err != nil {
		return m, err
	}
	ent2 := gotransit.Stop{}
	ent2.StopID = "bar"
	ent2.SetCoordinates([2]float64{-123.0, 42.0})
	ent2.FeedVersionID = m.FeedVersionID
	m.StopID1, err = adapter.Insert(&ent2)
	if err != nil {
		return m, err
	}
	ent3 := gotransit.Stop{}
	ent3.StopID = "foo"
	ent3.SetCoordinates([2]float64{-122.0, 43.0})
	ent3.FeedVersionID = m.FeedVersionID
	m.StopID2, err = adapter.Insert(&ent3)
	if err != nil {
		return m, err
	}
	return m, nil
}
