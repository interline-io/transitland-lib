package tlcsv

import (
	"math"
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

func TestGetString(t *testing.T) {
	ent := gtfs.StopTime{
		TripID:            tt.NewString("123"),
		StopID:            tt.NewString("456"),
		ArrivalTime:       tt.NewSeconds(3600),
		DepartureTime:     tt.NewSeconds(7200),
		ShapeDistTraveled: tt.NewFloat(123.456),
	}
	expect := map[string]string{
		"trip_id":             "123",
		"stop_id":             "456",
		"arrival_time":        "01:00:00",
		"departure_time":      "02:00:00",
		"shape_dist_traveled": "123.45600",
		"timepoint":           "",
	}
	for k, v := range expect {
		c, err := GetString(&ent, k)
		if err != nil {
			t.Error(err)
		}
		if c != v {
			t.Errorf("got %s expect %s", c, v)
		}
	}
}

func TestSetString(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		ent := gtfs.Frequency{}
		if err := SetString(&ent, "headway_secs", "123"); err != nil {
			t.Error(err)
		}
		if exp := 123; exp != ent.HeadwaySecs.Int() {
			t.Errorf("got %d expect %d", ent.HeadwaySecs.Val, exp)
		}
	})
	t.Run("string", func(t *testing.T) {
		ent := gtfs.Frequency{}
		if err := SetString(&ent, "trip_id", "123"); err != nil {
			t.Error(err)
		}
		if exp := "123"; exp != ent.TripID.Val {
			t.Errorf("got %s expect %s", ent.TripID, exp)
		}
	})
	t.Run("float", func(t *testing.T) {
		ent := gtfs.FareAttribute{}
		if err := SetString(&ent, "price", "123.456"); err != nil {
			t.Error(err)
		}
		if exp := 123.456; math.Abs(exp-ent.Price.Val) > 0.001 {
			t.Errorf("got %f expect %f", ent.Price.Val, exp)
		}
	})
	t.Run("time", func(t *testing.T) {
		ent := gtfs.Calendar{}
		if err := SetString(&ent, "start_date", "20190802"); err != nil {
			t.Error(err)
		}
		exp := []int{2019, 8, 02}
		got := []int{ent.StartDate.Year(), int(ent.StartDate.Month()), ent.StartDate.Day()}
		for i := 0; i < len(exp); i++ {
			if got[i] != exp[i] {
				t.Errorf("got %d expect %d", got[i], exp[i])
			}
		}

	})
	t.Run("widetime", func(t *testing.T) {
		ent := gtfs.Frequency{}
		if err := SetString(&ent, "start_time", "01:00:00"); err != nil {
			t.Error(err)
		}
		if exp := 3600; exp != ent.StartTime.Int() {
			t.Errorf("got %d expect %d", ent.StartTime.Int(), exp)
		}
	})
}
