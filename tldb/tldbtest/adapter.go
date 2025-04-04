package tldbtest

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
)

type Adapter = tldb.Adapter

var TestAdapters = map[string]func() Adapter{}

// Interface tests for Adapter
func AdapterTest(ctx context.Context, t *testing.T, adapter Adapter) {
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
		ent := gtfs.Trip{}
		ent.ID = m.TripID
		ent.TripHeadsign.Set(v)
		err = adapter.Update(ctx, &ent, "trip_headsign")
		if err != nil {
			t.Error(err)
		}
		ent2 := gtfs.Trip{}
		ent2.ID = m.TripID
		if err := adapter.Find(ctx, &ent2); err != nil {
			t.Error(err)
		}
		if ent2.TripHeadsign.Val != v {
			t.Errorf("got %s expected %s", ent2.TripHeadsign.Val, v)
		}
	})
	t.Run("Get", func(t *testing.T) {
		ent := gtfs.Trip{}
		ent.ID = m.TripID
		if err := adapter.Find(ctx, &ent); err != nil {
			t.Error(err)
		}
		if ent.ID != m.TripID {
			t.Errorf("got %d expected %d", ent.ID, m.TripID)
		}
	})
	t.Run("Select", func(t *testing.T) {
		ents := []gtfs.Stop{}
		if err := adapter.Select(ctx, &ents, "SELECT * FROM gtfs_stops WHERE id IN (?,?) AND feed_version_id = ? ORDER BY id ASC", m.StopID1, m.StopID2, m.FeedVersionID); err != nil {
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
	t.Run("TableExists", func(t *testing.T) {
		checkTable := "absolutely_does_not_exist"
		if ok, err := adapter.TableExists(checkTable); err != nil {
			t.Fatal(err)
		} else if ok {
			t.Errorf("expected table '%s' not to exist", checkTable)
		}
		doesExist := "feed_versions"
		if ok, err := adapter.TableExists(doesExist); err != nil {
			t.Fatal(err)
		} else if !ok {
			t.Errorf("expected table '%s' to exist", doesExist)
		}

	})
	t.Run("MultiInsert", func(t *testing.T) {
		st1 := gtfs.StopTime{}
		st1.FeedVersionID = m.FeedVersionID
		st1.StopID.Set(strconv.Itoa(m.StopID1))
		st1.TripID.Set(strconv.Itoa(m.TripID))
		st1.StopSequence.Set(1)
		st1.ArrivalTime = tt.NewSeconds(0)
		st1.DepartureTime = tt.NewSeconds(1)
		st2 := gtfs.StopTime{}
		st2.FeedVersionID = m.FeedVersionID
		st2.StopID.Set(strconv.Itoa(m.StopID2))
		st2.TripID.Set(strconv.Itoa(m.TripID))
		st2.StopSequence.Set(2)
		st2.ArrivalTime = tt.NewSeconds(2)
		st2.DepartureTime = tt.NewSeconds(3)
		sts := make([]interface{}, 0)
		sts = append(sts, &st1, &st2)
		if _, err := adapter.MultiInsert(ctx, sts); err != nil {
			t.Error(err)
		}
		sts2 := []gtfs.StopTime{}
		if err := adapter.Select(ctx, &sts2, "SELECT * FROM gtfs_stop_times WHERE feed_version_id = ? ORDER BY stop_sequence ASC", m.FeedVersionID); err != nil {
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
				t.Errorf("got '%d' expected '%d'", got1.StopSequence.Val, st1.StopSequence.Val)
			}
		}
	})
	t.Run("Tx Commit", func(t *testing.T) {
		// Check commit
		v := "Test Tx"
		ent := gtfs.Trip{}
		ent.ID = m.TripID
		ent.TripHeadsign.Set(v)
		adapter.Tx(func(atx Adapter) error {
			err := atx.Update(ctx, &ent, "trip_headsign")
			if err != nil {
				t.Error(err)
			}
			return err
		})
		ent2 := gtfs.Trip{}
		ent2.ID = m.TripID
		if err := adapter.Find(ctx, &ent2); err != nil {
			t.Error(err)
		}
		if ent2.TripHeadsign.Val != v {
			t.Errorf("got %s expected %s", ent2.TripHeadsign, v)
		}
	})
	t.Run("Tx Rollback", func(t *testing.T) {
		// Check rollback
		v := "Test Rollback"
		ent := gtfs.Trip{}
		ent.ID = m.TripID
		ent.TripHeadsign.Set(v)
		adapter.Tx(func(atx Adapter) error {
			err := atx.Update(ctx, &ent, "trip_headsign")
			if err != nil {
				t.Error(err)
			}
			return errors.New("rollback")
		})
		ent2 := gtfs.Trip{}
		ent2.ID = m.TripID
		if err := adapter.Find(ctx, &ent2); err != nil {
			t.Error(err)
		}
		if ent2.TripHeadsign.Val == v {
			t.Errorf("got %s expected != %s", ent2.TripHeadsign, v)
		}
	})
}

func createTestFeedVersion(adapter Adapter) (int, error) {
	// Create Feed, FeedVersion
	ctx := context.TODO()
	m := 0
	t := fmt.Sprintf("%d", time.Now().UnixNano())
	feed := dmfr.Feed{}
	feed.FeedID = t
	feedid, err := adapter.Insert(ctx, &feed)
	if err != nil {
		return m, err
	}
	feed.ID = feedid
	fv := dmfr.FeedVersion{}
	fv.SHA1 = t
	fv.FeedID = feed.ID
	fv.EarliestCalendarDate = tt.NewDate(time.Now())
	fv.LatestCalendarDate = tt.NewDate(time.Now())
	m, err = adapter.Insert(ctx, &fv)
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
	ctx := context.TODO()
	m := minEnts{}
	var err error
	m.FeedVersionID, err = createTestFeedVersion(adapter)
	if err != nil {
		return m, err
	}
	//
	ent0 := gtfs.Agency{}
	ent0.AgencyID.Set("ok")
	ent0.AgencyName.Set("ok")
	ent0.AgencyURL.Set("https://example.com")
	ent0.AgencyTimezone.Set("America/Los_Angeles")
	ent0.FeedVersionID = m.FeedVersionID
	m.AgencyID, err = adapter.Insert(ctx, &ent0)
	if err != nil {
		return m, err
	}
	ent4 := gtfs.Route{}
	ent4.RouteID.Set("ok")
	ent4.RouteType.Set(0)
	ent4.AgencyID.Set(strconv.Itoa(m.AgencyID))
	ent4.FeedVersionID = m.FeedVersionID
	m.RouteID, err = adapter.Insert(ctx, &ent4)
	if err != nil {
		return m, err
	}
	cal := gtfs.Calendar{}
	cal.StartDate.Set(time.Now())
	cal.EndDate.Set(time.Now())
	cal.ServiceID.Set("ok")
	cal.Monday.Set(0)
	cal.Tuesday.Set(0)
	cal.Wednesday.Set(0)
	cal.Thursday.Set(0)
	cal.Friday.Set(0)
	cal.Saturday.Set(0)
	cal.Sunday.Set(0)
	cal.Generated.Set(false)
	cal.FeedVersionID = m.FeedVersionID
	m.ServiceID, err = adapter.Insert(ctx, &cal)
	if err != nil {
		return m, err
	}
	ent1 := gtfs.Trip{}
	ent1.TripID.Set("ok")
	ent1.RouteID.Set(strconv.Itoa(m.RouteID))
	ent1.ServiceID.Set(strconv.Itoa(m.ServiceID))
	ent1.DirectionID.Set(0)
	ent1.StopPatternID.Set(0)
	ent1.JourneyPatternID.Set("")
	ent1.JourneyPatternOffset.Set(0)
	ent1.FeedVersionID = m.FeedVersionID
	m.TripID, err = adapter.Insert(ctx, &ent1)
	if err != nil {
		return m, err
	}
	ent2 := gtfs.Stop{}
	ent2.StopID.Set("bar")
	ent2.SetCoordinates([2]float64{-123.0, 42.0})
	ent2.FeedVersionID = m.FeedVersionID
	ent2.LocationType.Set(0)
	m.StopID1, err = adapter.Insert(ctx, &ent2)
	if err != nil {
		return m, err
	}
	ent3 := gtfs.Stop{}
	ent3.StopID.Set("foo")
	ent3.SetCoordinates([2]float64{-122.0, 43.0})
	ent3.FeedVersionID = m.FeedVersionID
	ent3.LocationType.Set(0)
	m.StopID2, err = adapter.Insert(ctx, &ent3)
	if err != nil {
		return m, err
	}
	return m, nil
}
