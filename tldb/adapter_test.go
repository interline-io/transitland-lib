package tldb

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/tl"
)

var testAdapters = map[string]func() Adapter{}

// Interface tests for Adapter
func testAdapter(t *testing.T, adapter Adapter) {
	if err := adapter.Open(); err != nil {
		t.Error(err)
	}
	if err := adapter.Create(); err != nil {
		t.Error(err)
	}
	//
	var err error
	var m minEnts
	//
	t.Run("Insert", func(t *testing.T) {
		var err error
		// createMinEntities uses Insert
		m, err = createMinEntities(adapter)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	})
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	t.Run("Update", func(t *testing.T) {
		v := "Test Update"
		ent := tl.Trip{}
		ent.ID = m.TripID
		ent.TripHeadsign = v
		err = adapter.Update(&ent, "trip_headsign")
		if err != nil {
			t.Error(err)
		}
		ent2 := tl.Trip{}
		ent2.ID = m.TripID
		if err := adapter.Find(&ent2); err != nil {
			t.Error(err)
		}
		if ent2.TripHeadsign != v {
			t.Errorf("got %s expected %s", ent2.TripHeadsign, v)
		}
	})
	t.Run("Get", func(t *testing.T) {
		ent := tl.Trip{}
		ent.ID = m.TripID
		if err := adapter.Find(&ent); err != nil {
			t.Error(err)
		}
		if ent.ID != m.TripID {
			t.Errorf("got %d expected %d", ent.ID, m.TripID)
		}
	})
	t.Run("Select", func(t *testing.T) {
		ents := []tl.Stop{}
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
	t.Run("CopyInsert", func(t *testing.T) {
		st1 := tl.StopTime{}
		st1.FeedVersionID = m.FeedVersionID
		st1.StopID = strconv.Itoa(m.StopID1)
		st1.TripID = strconv.Itoa(m.TripID)
		st1.StopSequence = 1
		st1.ArrivalTime = 0
		st1.DepartureTime = 1
		st2 := tl.StopTime{}
		st2.FeedVersionID = m.FeedVersionID
		st2.StopID = strconv.Itoa(m.StopID2)
		st2.TripID = strconv.Itoa(m.TripID)
		st2.StopSequence = 2
		st2.ArrivalTime = 2
		st2.DepartureTime = 3
		sts := make([]interface{}, 0)
		sts = append(sts, &st1, &st2)
		if err := adapter.CopyInsert(sts); err != nil {
			t.Error(err)
		}
		sts2 := []tl.StopTime{}
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
	t.Run("Tx Commit", func(t *testing.T) {
		// Check commit
		v := "Test Tx"
		ent := tl.Trip{}
		ent.ID = m.TripID
		ent.TripHeadsign = v
		adapter.Tx(func(atx Adapter) error {
			err := atx.Update(&ent, "trip_headsign")
			if err != nil {
				t.Error(err)
			}
			return err
		})
		ent2 := tl.Trip{}
		ent2.ID = m.TripID
		if err := adapter.Find(&ent2); err != nil {
			t.Error(err)
		}
		if ent2.TripHeadsign != v {
			t.Errorf("got %s expected %s", ent2.TripHeadsign, v)
		}
	})
	t.Run("Tx Rollback", func(t *testing.T) {
		// Check rollback
		v := "Test Rollback"
		ent := tl.Trip{}
		ent.ID = m.TripID
		ent.TripHeadsign = v
		adapter.Tx(func(atx Adapter) error {
			err := atx.Update(&ent, "trip_headsign")
			if err != nil {
				t.Error(err)
			}
			return errors.New("rollback")
		})
		ent2 := tl.Trip{}
		ent2.ID = m.TripID
		if err := adapter.Find(&ent2); err != nil {
			t.Error(err)
		}
		if ent2.TripHeadsign == v {
			t.Errorf("got %s expected != %s", ent2.TripHeadsign, v)
		}
	})
}

func createTestFeedVersion(adapter Adapter) (int, error) {
	// Create Feed, FeedVersion
	m := 0
	t := fmt.Sprintf("%d", time.Now().UnixNano())
	feed := tl.Feed{}
	feed.FeedID = t
	feedid, err := adapter.Insert(&feed)
	if err != nil {
		return m, err
	}
	feed.ID = feedid
	fv := tl.FeedVersion{}
	fv.SHA1 = t
	fv.FeedID = feed.ID
	m, err = adapter.Insert(&fv)
	return m, err
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
func createMinEntities(adapter Adapter) (minEnts, error) {
	m := minEnts{}
	var err error
	m.FeedVersionID, err = createTestFeedVersion(adapter)
	if err != nil {
		return m, err
	}
	//
	ent0 := tl.Agency{}
	ent0.AgencyID = "ok"
	ent0.FeedVersionID = m.FeedVersionID
	m.AgencyID, err = adapter.Insert(&ent0)
	if err != nil {
		return m, err
	}
	ent4 := tl.Route{}
	ent4.RouteID = "ok"
	ent4.AgencyID = strconv.Itoa(m.AgencyID)
	ent4.FeedVersionID = m.FeedVersionID
	m.RouteID, err = adapter.Insert(&ent4)
	if err != nil {
		return m, err
	}
	cal := tl.Calendar{}
	cal.StartDate = time.Now()
	cal.EndDate = time.Now()
	cal.ServiceID = "ok"
	cal.FeedVersionID = m.FeedVersionID
	m.ServiceID, err = adapter.Insert(&cal)
	if err != nil {
		return m, err
	}
	ent1 := tl.Trip{}
	ent1.TripID = "ok"
	ent1.RouteID = strconv.Itoa(m.RouteID)
	ent1.ServiceID = strconv.Itoa(m.ServiceID)
	ent1.FeedVersionID = m.FeedVersionID
	m.TripID, err = adapter.Insert(&ent1)
	if err != nil {
		return m, err
	}
	ent2 := tl.Stop{}
	ent2.StopID = "bar"
	ent2.SetCoordinates([2]float64{-123.0, 42.0})
	ent2.FeedVersionID = m.FeedVersionID
	m.StopID1, err = adapter.Insert(&ent2)
	if err != nil {
		return m, err
	}
	ent3 := tl.Stop{}
	ent3.StopID = "foo"
	ent3.SetCoordinates([2]float64{-122.0, 43.0})
	ent3.FeedVersionID = m.FeedVersionID
	m.StopID2, err = adapter.Insert(&ent3)
	if err != nil {
		return m, err
	}
	return m, nil
}
