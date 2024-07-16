package tt

import (
	"errors"
	"time"
)

// Allow for time mocking
type Clock interface {
	Now() time.Time
}

// Real system clock
type RealClock struct{}

func (dc RealClock) Now() time.Time {
	return time.Now().In(time.UTC)
}

// A mock clock with a fixed time
type MockClock struct {
	T time.Time
}

func (dc MockClock) Now() time.Time {
	return dc.T
}

func TimeAt(date string, wt string, tz string, startDate string, endDate string, fallbackWeek string, useFallback bool) (time.Time, error) {
	return timeAtClock(date, wt, tz, startDate, endDate, fallbackWeek, useFallback, RealClock{})
}

func timeAtClock(date string, wt string, tz string, startDate string, endDate string, fallbackWeek string, useFallback bool, cl Clock) (time.Time, error) {
	// Get timezone
	baseTime := time.Time{}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return baseTime, err
	}

	// Current date
	clNow := cl.Now().In(loc)
	clDay := clNow.Day()
	clYear := clNow.Year()
	clMonth := clNow.Month()
	wtHour := clNow.Hour()
	wtMin := clNow.Minute()
	wtSec := clNow.Second()

	// Get local HMS
	if wt != "" {
		t, err := NewWideTime(wt)
		if err != nil {
			return baseTime, err
		}
		seconds := t.Seconds
		wtHour = seconds / 3600
		wtMin = (seconds % 3600) / 60
		wtSec = (seconds % 60)
	}

	// Parse date or use special label
	dowOffset := -1
	switch date {
	case "now":
		// default
	case "":
		// equiv to "now"
	case "next-sunday":
		dowOffset = 0
	case "next-monday":
		dowOffset = 1
	case "next-tuesday":
		dowOffset = 2
	case "next-wednesday":
		dowOffset = 3
	case "next-thursday":
		dowOffset = 4
	case "next-friday":
		dowOffset = 5
	case "next-saturday":
		dowOffset = 6
	default:
		// Update to parsed YYYY-MM-DD
		t, err := time.Parse("2006-01-02", date)
		if err != nil {
			return baseTime, errors.New("could not parse date")
		}
		clYear = t.Year()
		clMonth = t.Month()
		clDay = t.Day()
	}

	// Construct time from parsed components
	baseTime = time.Date(clYear, clMonth, clDay, wtHour, wtMin, wtSec, 0, loc)

	// Check the next 7 days to get the correct weekday
	if dowOffset >= 0 {
		dowTime := baseTime
		for i := 0; i < 7; i++ {
			curDow := dowTime.Weekday()
			if int(curDow) == dowOffset {
				baseTime = dowTime
				break
			}
			dowTime = dowTime.AddDate(0, 0, 1)
		}
	}

	// Check bounds
	if useFallback {
		startTime, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			return baseTime, errors.New("could not parse start time")
		}
		endTime, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			return baseTime, errors.New("could not parse end time")
		}
		fallbackTime, err := time.Parse("2006-01-02", fallbackWeek)
		if err != nil {
			return baseTime, errors.New("could not parse fallback week")
		}
		startTime = midnight(startTime, loc)
		endTime = midnight(endTime, loc).AddDate(0, 0, 1)
		if endTime.Before(startTime) {
			return baseTime, errors.New("end time before start time")
		}
		if baseTime.Before(startTime) || baseTime.After(endTime) {
			// fmt.Println("using fallback: ", fallbackTime, "date:", baseTime, "bounds:", startTime, endTime)
			ft := midnight(fallbackTime, loc)
			for i := 0; i < 7; i++ {
				if ft.Weekday() == baseTime.Weekday() {
					baseTime = time.Date(ft.Year(), ft.Month(), ft.Day(), baseTime.Hour(), baseTime.Minute(), baseTime.Second(), 0, loc)
					break
				}
				ft = ft.AddDate(0, 0, 1)
			}
		}
	}
	return baseTime, nil
}

func midnight(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}
