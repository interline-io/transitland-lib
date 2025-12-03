package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

func TestLocationGroup(t *testing.T) {
	lg := LocationGroup{
		LocationGroupID:   tt.NewString("lg1"),
		LocationGroupName: tt.NewString("Downtown"),
	}
	assert.Equal(t, "lg1", lg.EntityKey())
	assert.Equal(t, "location_groups.txt", lg.Filename())
	assert.Equal(t, "gtfs_location_groups", lg.TableName())
}

func TestLocationGroupStop(t *testing.T) {
	lgs := LocationGroupStop{
		LocationGroupID: tt.NewKey("lg1"),
		StopID:          tt.NewKey("stop1"),
	}
	assert.Equal(t, "location_group_stops.txt", lgs.Filename())
	assert.Equal(t, "gtfs_location_group_stops", lgs.TableName())
}

func TestBookingRule(t *testing.T) {
	t.Run("booking_type=0", func(t *testing.T) {
		br := BookingRule{
			BookingRuleID: tt.NewString("rule1"),
			BookingType:   tt.NewInt(0),
		}
		assert.Equal(t, "rule1", br.EntityKey())
		assert.Equal(t, "booking_rules.txt", br.Filename())
		assert.Equal(t, "gtfs_booking_rules", br.TableName())

		// booking_type=0: prior_notice_duration_max forbidden
		errs := br.ConditionalErrors()
		assert.Len(t, errs, 0)

		br.PriorNoticeDurationMax = tt.NewInt(30)
		errs = br.ConditionalErrors()
		assert.Greater(t, len(errs), 0)
	})

	t.Run("booking_type=1", func(t *testing.T) {
		br := BookingRule{
			BookingRuleID: tt.NewString("rule2"),
			BookingType:   tt.NewInt(1),
		}

		// booking_type=1: prior_notice_duration_min required
		errs := br.ConditionalErrors()
		assert.Greater(t, len(errs), 0)

		br.PriorNoticeDurationMin = tt.NewInt(15)
		errs = br.ConditionalErrors()
		assert.Len(t, errs, 0)
	})

	t.Run("booking_type=2", func(t *testing.T) {
		br := BookingRule{
			BookingRuleID: tt.NewString("rule3"),
			BookingType:   tt.NewInt(2),
		}

		// booking_type=2: prior_notice_last_day required
		errs := br.ConditionalErrors()
		assert.Greater(t, len(errs), 0)

		br.PriorNoticeLastDay = tt.NewInt(1)
		errs = br.ConditionalErrors()
		assert.Len(t, errs, 0)

		// booking_type=2: prior_notice_duration_max forbidden
		br.PriorNoticeDurationMax = tt.NewInt(30)
		errs = br.ConditionalErrors()
		assert.Greater(t, len(errs), 0)
	})
}

func TestLocation(t *testing.T) {
	loc := Location{
		LocationID: tt.NewString("loc1"),
		StopName:   tt.NewString("Flexible Zone"),
		StopDesc:   tt.NewString("On-demand service area"),
	}
	assert.Equal(t, "loc1", loc.EntityKey())
	assert.Equal(t, "locations.geojson", loc.Filename())
	assert.Equal(t, "gtfs_locations", loc.TableName())

	// Geometry is required
	errs := loc.ConditionalErrors()
	assert.Greater(t, len(errs), 0)
}

func TestStopTimeFlexFields(t *testing.T) {
	st := StopTime{
		TripID:                   tt.NewString("trip1"),
		StopID:                   tt.NewKey("stop1"),
		StopSequence:             tt.NewInt(1),
		StartPickupDropOffWindow: tt.NewSeconds(28800), // 8:00:00
		EndPickupDropOffWindow:   tt.NewSeconds(32400), // 9:00:00
		PickupBookingRuleID:      tt.NewKey("rule1"),
		MeanDurationFactor:       tt.NewFloat(1.5),
		SafeDurationFactor:       tt.NewFloat(2.0),
	}

	// Verify fields can be set and retrieved
	val, err := st.GetString("start_pickup_drop_off_window")
	assert.NoError(t, err)
	assert.Equal(t, "08:00:00", val)

	val, err = st.GetString("pickup_booking_rule_id")
	assert.NoError(t, err)
	assert.Equal(t, "rule1", val)

	val, err = st.GetString("mean_duration_factor")
	assert.NoError(t, err)
	assert.Contains(t, val, "1.5")
}
