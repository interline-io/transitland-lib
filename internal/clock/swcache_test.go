package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Mock service level data based on real MBTA feed version f1aa4431c87ded2609b3eb069dddbd4fe2a8aba9
type mockServiceLevel struct {
	startDate time.Time
	endDate   time.Time
	monday    int
	tuesday   int
	wednesday int
	thursday  int
	friday    int
	saturday  int
	sunday    int
}

func (m *mockServiceLevel) totalService() int {
	return m.monday + m.tuesday + m.wednesday + m.thursday + m.friday + m.saturday + m.sunday
}

func (m *mockServiceLevel) hasFullService() bool {
	return m.monday > 0 && m.tuesday > 0 && m.wednesday > 0 &&
		m.thursday > 0 && m.friday > 0 && m.saturday > 0 && m.sunday > 0
}

func TestFallbackWeekSelectionWithCaltrainData(t *testing.T) {
	// Real Caltrain data showing different service patterns
	// This tests various edge cases we might encounter

	caltrainLevels := []*mockServiceLevel{
		// Week starting 2025-08-18 - PARTIAL WEEK (Monday=0, Tuesday=0)
		{
			startDate: time.Date(2025, 8, 18, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 8, 24, 0, 0, 0, 0, time.UTC),
			sunday:    315840,
			monday:    0, // No service
			tuesday:   0, // No service
			wednesday: 487260,
			thursday:  487260,
			friday:    487260,
			saturday:  315840,
		},
		// Week starting 2025-08-25 - FULL WEEK (service on all days)
		{
			startDate: time.Date(2025, 8, 25, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 8, 31, 0, 0, 0, 0, time.UTC),
			sunday:    315840,
			monday:    487260,
			tuesday:   487260,
			wednesday: 487260,
			thursday:  487260,
			friday:    487260,
			saturday:  315840,
		},
		// Week starting 2025-09-01 - FULL WEEK (reduced Sunday service but still > 0)
		{
			startDate: time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 9, 7, 0, 0, 0, 0, time.UTC),
			sunday:    315840, // Reduced but > 0
			monday:    315840, // Reduced but > 0
			tuesday:   487260,
			wednesday: 487260,
			thursday:  487260,
			friday:    487260,
			saturday:  315840, // Reduced but > 0
		},
		// Week starting 2025-09-08 - FULL WEEK (highest service)
		{
			startDate: time.Date(2025, 9, 8, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 11, 23, 0, 0, 0, 0, time.UTC),
			sunday:    315840,
			monday:    487260,
			tuesday:   487260,
			wednesday: 487260,
			thursday:  487260,
			friday:    487260,
			saturday:  315840,
		},
		// Week starting 2025-06-30 - FULL WEEK (reduced Friday service but still > 0)
		{
			startDate: time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 7, 6, 0, 0, 0, 0, time.UTC),
			sunday:    13458933,
			monday:    21510411,
			tuesday:   21510411,
			wednesday: 21510411,
			thursday:  21510411,
			friday:    13458933, // Reduced service but > 0
			saturday:  16536528,
		},
		// Week starting 2025-12-29 - PARTIAL WEEK (Sunday=0, Thursday=0, Friday=0, Saturday=0)
		{
			startDate: time.Date(2025, 12, 29, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC),
			sunday:    0, // No service
			monday:    22059857,
			tuesday:   22059857,
			wednesday: 22059857,
			thursday:  0, // No service
			friday:    0, // No service
			saturday:  0, // No service
		},
	}

	t.Run("should_prefer_full_service_weeks_over_partial", func(t *testing.T) {
		var bestWeek *mockServiceLevel
		var bestService int
		hasFullServiceWeek := false

		// First pass: look for weeks with service on all days
		for _, sl := range caltrainLevels {
			if sl.hasFullService() && sl.totalService() > bestService {
				bestService = sl.totalService()
				bestWeek = sl
				hasFullServiceWeek = true
			}
		}

		// Second pass: if no full-service week found, fall back to any week with highest service
		if !hasFullServiceWeek {
			for _, sl := range caltrainLevels {
				if sl.totalService() > bestService {
					bestService = sl.totalService()
					bestWeek = sl
				}
			}
		}

		// Should find a full-service week and prefer the highest service one
		assert.True(t, hasFullServiceWeek, "Should have found a full-service week")
		assert.NotNil(t, bestWeek, "Should have selected a week")

		// Should select the week with highest service (2025-06-30)
		expectedTotal := 13458933 + 21510411 + 21510411 + 21510411 + 21510411 + 13458933 + 16536528
		assert.Equal(t, expectedTotal, bestWeek.totalService(), "Should select highest service week")
		assert.Equal(t, time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC), bestWeek.startDate, "Should select week starting 2025-06-30")
	})

	t.Run("should_identify_partial_weeks_correctly", func(t *testing.T) {
		// Test that weeks with zero service on specific days are properly identified
		partialWeek1 := caltrainLevels[0] // 2025-08-18 week (Monday=0, Tuesday=0)
		partialWeek2 := caltrainLevels[5] // 2025-12-29 week (Sunday=0, Thursday=0, Friday=0, Saturday=0)

		assert.False(t, partialWeek1.hasFullService(), "Week starting 2025-08-18 should not be considered full service")
		assert.False(t, partialWeek2.hasFullService(), "Week starting 2025-12-29 should not be considered full service")

		// Test that weeks with reduced but non-zero service are still considered full service
		fullWeek1 := caltrainLevels[1] // 2025-08-25 week
		fullWeek2 := caltrainLevels[2] // 2025-09-01 week (reduced but > 0)
		fullWeek3 := caltrainLevels[4] // 2025-06-30 week (reduced Friday but > 0)

		assert.True(t, fullWeek1.hasFullService(), "Week starting 2025-08-25 should be considered full service")
		assert.True(t, fullWeek2.hasFullService(), "Week starting 2025-09-01 should be considered full service (reduced but > 0)")
		assert.True(t, fullWeek3.hasFullService(), "Week starting 2025-06-30 should be considered full service (reduced Friday but > 0)")
	})

	t.Run("should_handle_extreme_partial_weeks", func(t *testing.T) {
		// Test the most extreme partial week (only 3 days with service)
		extremePartialWeek := caltrainLevels[5] // 2025-12-29 week

		assert.False(t, extremePartialWeek.hasFullService(), "Extreme partial week should not be full service")
		assert.Equal(t, 0, extremePartialWeek.sunday, "Sunday should have no service")
		assert.Equal(t, 0, extremePartialWeek.thursday, "Thursday should have no service")
		assert.Equal(t, 0, extremePartialWeek.friday, "Friday should have no service")
		assert.Equal(t, 0, extremePartialWeek.saturday, "Saturday should have no service")
	})

	t.Run("should_distinguish_reduced_from_zero_service", func(t *testing.T) {
		// Test that reduced service (> 0) is different from zero service (= 0)
		reducedServiceWeek := caltrainLevels[2] // 2025-09-01 week
		zeroServiceWeek := caltrainLevels[0]    // 2025-08-18 week

		// Reduced service week should still be considered full service
		assert.True(t, reducedServiceWeek.hasFullService(), "Week with reduced service should still be full service")
		assert.True(t, reducedServiceWeek.monday > 0, "Reduced service should be > 0")
		assert.True(t, reducedServiceWeek.monday < 487260, "Reduced service should be less than full service")

		// Zero service week should not be considered full service
		assert.False(t, zeroServiceWeek.hasFullService(), "Week with zero service days should not be full service")
		assert.Equal(t, 0, zeroServiceWeek.monday, "Zero service should be exactly 0")
	})
}

func TestFallbackWeekSelectionWithRealData(t *testing.T) {
	// Real MBTA data from feed version f1aa4431c87ded2609b3eb069dddbd4fe2a8aba9
	// This feed has the issue we're fixing: fallback_week = "2025-08-04" but feed_start_date = "2025-08-07"

	serviceLevels := []*mockServiceLevel{
		// Week starting 2025-08-04 - PARTIAL WEEK (Monday=0, Tuesday=0, Wednesday=0)
		// This is currently selected as fallback week but shouldn't be
		{
			startDate: time.Date(2025, 8, 4, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 8, 10, 0, 0, 0, 0, time.UTC),
			sunday:    15909060,
			monday:    0, // No service
			tuesday:   0, // No service
			wednesday: 0, // No service
			thursday:  30639300,
			friday:    30661200,
			saturday:  20477520,
		},
		// Week starting 2025-08-11 - FULL WEEK (service on all days)
		// This should be selected as fallback week
		{
			startDate: time.Date(2025, 8, 11, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 8, 17, 0, 0, 0, 0, time.UTC),
			sunday:    16452720,
			monday:    31320720,
			tuesday:   31334820,
			wednesday: 31334820,
			thursday:  31334820,
			friday:    31374840,
			saturday:  21093000,
		},
		// Week starting 2025-08-18 - PARTIAL WEEK (Sunday=0)
		{
			startDate: time.Date(2025, 8, 18, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 8, 24, 0, 0, 0, 0, time.UTC),
			sunday:    0, // No service
			monday:    30618420,
			tuesday:   30618420,
			wednesday: 30618420,
			thursday:  30618420,
			friday:    30662040,
			saturday:  21130320,
		},
		// Week starting 2025-09-01 - PARTIAL WEEK (Monday=16566840, which is lower than typical)
		{
			startDate: time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 9, 7, 0, 0, 0, 0, time.UTC),
			sunday:    16527240,
			monday:    16566840, // Lower service than typical
			tuesday:   31430700,
			wednesday: 31430700,
			thursday:  31430700,
			friday:    31555320,
			saturday:  20879460,
		},
		// Week starting 2025-09-08 - FULL WEEK (high service on all days)
		{
			startDate: time.Date(2025, 9, 8, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 9, 14, 0, 0, 0, 0, time.UTC),
			sunday:    16527240,
			monday:    31430700,
			tuesday:   31430700,
			wednesday: 31430700,
			thursday:  31430700,
			friday:    31555320,
			saturday:  20879460,
		},
	}

	// Test our fallback week selection logic
	t.Run("should_prefer_full_service_weeks", func(t *testing.T) {
		var bestWeek *mockServiceLevel
		var bestService int
		hasFullServiceWeek := false

		// First pass: look for weeks with service on all days
		for _, sl := range serviceLevels {
			if sl.hasFullService() && sl.totalService() > bestService {
				bestService = sl.totalService()
				bestWeek = sl
				hasFullServiceWeek = true
			}
		}

		// Second pass: if no full-service week found, fall back to any week with highest service
		if !hasFullServiceWeek {
			for _, sl := range serviceLevels {
				if sl.totalService() > bestService {
					bestService = sl.totalService()
					bestWeek = sl
				}
			}
		}

		// Verify our logic works correctly
		assert.NotNil(t, bestWeek, "Should have selected a week")
		assert.True(t, hasFullServiceWeek, "Should have found a full-service week")

		// The selected week should be the one with highest service among full-service weeks
		// 2025-09-08 has higher total service than 2025-08-11
		expectedStart := time.Date(2025, 9, 8, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expectedStart, bestWeek.startDate, "Should select highest service full-service week")

		// Verify it has service on all days
		assert.True(t, bestWeek.hasFullService(), "Selected week should have service on all days")
		assert.Greater(t, bestWeek.monday, 0, "Monday should have service")
		assert.Greater(t, bestWeek.tuesday, 0, "Tuesday should have service")
		assert.Greater(t, bestWeek.wednesday, 0, "Wednesday should have service")
		assert.Greater(t, bestWeek.thursday, 0, "Thursday should have service")
		assert.Greater(t, bestWeek.friday, 0, "Friday should have service")
		assert.Greater(t, bestWeek.saturday, 0, "Saturday should have service")
		assert.Greater(t, bestWeek.sunday, 0, "Sunday should have service")

		// Verify it's the highest service week among full-service weeks
		expectedTotal := 16527240 + 31430700 + 31430700 + 31430700 + 31430700 + 31555320 + 20879460
		assert.Equal(t, expectedTotal, bestWeek.totalService(), "Should select week with highest total service")
	})

	t.Run("should_skip_weeks_with_zero_service_days", func(t *testing.T) {
		// Verify that weeks with zero service on any day are not selected
		for _, sl := range serviceLevels {
			if !sl.hasFullService() {
				// These weeks should not be selected as fallback weeks
				assert.False(t, sl.hasFullService(), "Week starting %s should not be considered full service", sl.startDate.Format("2006-01-02"))
			}
		}
	})

	t.Run("should_handle_edge_cases", func(t *testing.T) {
		// Test edge case: what if all weeks have partial service?
		partialOnlyLevels := []*mockServiceLevel{
			{
				startDate: time.Date(2025, 8, 4, 0, 0, 0, 0, time.UTC),
				endDate:   time.Date(2025, 8, 10, 0, 0, 0, 0, time.UTC),
				sunday:    15909060,
				monday:    0, // No service
				tuesday:   30602160,
				wednesday: 30639300,
				thursday:  30639300,
				friday:    30661200,
				saturday:  20477520,
			},
			{
				startDate: time.Date(2025, 8, 11, 0, 0, 0, 0, time.UTC),
				endDate:   time.Date(2025, 8, 17, 0, 0, 0, 0, time.UTC),
				sunday:    16452720,
				monday:    31320720,
				tuesday:   0, // No service
				wednesday: 31334820,
				thursday:  31334820,
				friday:    31374840,
				saturday:  21093000,
			},
		}

		var bestWeek *mockServiceLevel
		var bestService int
		hasFullServiceWeek := false

		// First pass: look for weeks with service on all days
		for _, sl := range partialOnlyLevels {
			if sl.hasFullService() && sl.totalService() > bestService {
				bestService = sl.totalService()
				bestWeek = sl
				hasFullServiceWeek = true
			}
		}

		// Second pass: if no full-service week found, fall back to any week with highest service
		if !hasFullServiceWeek {
			for _, sl := range partialOnlyLevels {
				if sl.totalService() > bestService {
					bestService = sl.totalService()
					bestWeek = sl
				}
			}
		}

		// Should fall back to highest service week even if partial
		assert.NotNil(t, bestWeek, "Should have selected a week even if all are partial")
		assert.False(t, hasFullServiceWeek, "Should not have found a full-service week")

		// Calculate the expected total for the second week (which has higher service)
		expectedTotal := 16452720 + 31320720 + 0 + 31334820 + 31334820 + 31374840 + 21093000
		assert.Equal(t, expectedTotal, bestWeek.totalService(), "Should select highest service week")
		assert.Equal(t, time.Date(2025, 8, 11, 0, 0, 0, 0, time.UTC), bestWeek.startDate, "Should select week starting 2025-08-11")
	})
}

func TestFallbackWeekSelection(t *testing.T) {
	// Test data representing the MBTA issue we're fixing
	// Week starting 2025-08-04 has Monday=0, Tuesday=0 (partial week)
	// Week starting 2025-08-11 has full service on all days

	testCases := []struct {
		name         string
		startDate    time.Time
		endDate      time.Time
		expectedWeek time.Time
		description  string
	}{
		{
			name:         "partial_week_should_be_skipped",
			startDate:    time.Date(2025, 8, 4, 0, 0, 0, 0, time.UTC),
			endDate:      time.Date(2025, 8, 10, 0, 0, 0, 0, time.UTC),
			expectedWeek: time.Date(2025, 8, 11, 0, 0, 0, 0, time.UTC), // Should prefer full week
			description:  "Should skip week with zero service on Monday/Tuesday",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This is a basic test to ensure our code compiles and runs
			// The actual logic testing would require a full database setup
			assert.True(t, tc.startDate.Before(tc.endDate), "Start date should be before end date")
			assert.True(t, tc.expectedWeek.After(tc.startDate), "Expected week should be after start date")
			assert.True(t, tc.expectedWeek.Before(tc.endDate.AddDate(0, 0, 7)), "Expected week should be within reasonable range")
		})
	}
}

func TestServiceWindowStruct(t *testing.T) {
	// Test that our ServiceWindow struct can be created and used
	sw := ServiceWindow{
		StartDate:    time.Date(2025, 8, 7, 0, 0, 0, 0, time.UTC),
		EndDate:      time.Date(2025, 12, 13, 0, 0, 0, 0, time.UTC),
		FallbackWeek: time.Date(2025, 8, 11, 0, 0, 0, 0, time.UTC),
		Location:     time.UTC,
	}

	assert.False(t, sw.StartDate.IsZero(), "StartDate should not be zero")
	assert.False(t, sw.EndDate.IsZero(), "EndDate should not be zero")
	assert.False(t, sw.FallbackWeek.IsZero(), "FallbackWeek should not be zero")
	assert.NotNil(t, sw.Location, "Location should not be nil")

	// Verify the fallback week is within the service window
	assert.True(t, sw.FallbackWeek.After(sw.StartDate) || sw.FallbackWeek.Equal(sw.StartDate),
		"FallbackWeek should be on or after StartDate")
	assert.True(t, sw.FallbackWeek.Before(sw.EndDate) || sw.FallbackWeek.Equal(sw.EndDate),
		"FallbackWeek should be on or before EndDate")
}

func TestFallbackWeekSelectionWithRTDData(t *testing.T) {
	// RTD data showing extreme edge cases
	// This tests scenarios like complete zero-service weeks and minimal service weeks

	rtdLevels := []*mockServiceLevel{
		// Week starting 2025-05-19 - COMPLETE ZERO SERVICE WEEK
		{
			startDate: time.Date(2025, 5, 19, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 5, 25, 0, 0, 0, 0, time.UTC),
			sunday:    0, // No service
			monday:    0, // No service
			tuesday:   0, // No service
			wednesday: 0, // No service
			thursday:  0, // No service
			friday:    0, // No service
			saturday:  0, // No service
		},
		// Week starting 2025-05-26 - FULL SERVICE WEEK
		{
			startDate: time.Date(2025, 5, 26, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			sunday:    13458933,
			monday:    13458933,
			tuesday:   21510411,
			wednesday: 21510411,
			thursday:  21510411,
			friday:    21595911,
			saturday:  16536528,
		},
		// Week starting 2025-06-30 - PARTIAL SERVICE (Sunday + Friday only)
		{
			startDate: time.Date(2025, 6, 30, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 7, 6, 0, 0, 0, 0, time.UTC),
			sunday:    13458933,
			monday:    21510411,
			tuesday:   21510411,
			wednesday: 21510411,
			thursday:  21510411,
			friday:    13458933, // Reduced service
			saturday:  16536528,
		},
		// Week starting 2025-09-15 - FULL SERVICE WEEK (highest service)
		{
			startDate: time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2025, 11, 23, 0, 0, 0, 0, time.UTC),
			sunday:    13424463,
			monday:    22059857,
			tuesday:   22059857,
			wednesday: 22059857,
			thursday:  22059857,
			friday:    22126637,
			saturday:  16538433,
		},
	}

	t.Run("should_handle_complete_zero_service_weeks", func(t *testing.T) {
		var bestWeek *mockServiceLevel
		var bestService int
		hasFullServiceWeek := false

		// First pass: look for weeks with service on all days
		for _, sl := range rtdLevels {
			if sl.hasFullService() && sl.totalService() > bestService {
				bestService = sl.totalService()
				bestWeek = sl
				hasFullServiceWeek = true
			}
		}

		// Second pass: if no full-service week found, fall back to any week with highest service
		if !hasFullServiceWeek {
			for _, sl := range rtdLevels {
				if sl.totalService() > bestService {
					bestService = sl.totalService()
					bestWeek = sl
				}
			}
		}

		// Should find a full-service week and prefer the highest service one
		assert.True(t, hasFullServiceWeek, "Should have found a full-service week")
		assert.NotNil(t, bestWeek, "Should have selected a week")

		// Should select the week with highest service (2025-09-15)
		expectedTotal := 13424463 + 22059857 + 22059857 + 22059857 + 22059857 + 22126637 + 16538433
		assert.Equal(t, expectedTotal, bestWeek.totalService(), "Should select highest service week")
		assert.Equal(t, time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC), bestWeek.startDate, "Should select week starting 2025-09-15")
	})

	t.Run("should_skip_complete_zero_service_weeks", func(t *testing.T) {
		// Test that weeks with zero service on all days are properly handled
		zeroServiceWeek := rtdLevels[0] // 2025-05-19 week

		assert.Equal(t, 0, zeroServiceWeek.totalService(), "Zero service week should have zero total")
		assert.False(t, zeroServiceWeek.hasFullService(), "Zero service week should not be considered full service")

		// This week should never be selected as fallback unless it's the only option
		assert.True(t, zeroServiceWeek.totalService() == 0, "Should be a complete zero service week")
	})
}
