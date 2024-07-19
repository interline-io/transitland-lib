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

// RelativeDate gets a date reltive to the provided time; currentTime should have timezone set.
func RelativeDate(currentTime time.Time, relativeDateLabel string) (time.Time, error) {
	// Current date
	clNow := currentTime
	clDay := clNow.Day()
	clYear := clNow.Year()
	clMonth := clNow.Month()
	clHour := clNow.Hour()
	clMin := clNow.Minute()
	clSec := clNow.Second()
	clLoc := clNow.Location()

	// Parse date or use special label
	dowOffset := -1
	switch relativeDateLabel {
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
		t, err := time.Parse("2006-01-02", relativeDateLabel)
		if err != nil {
			return time.Time{}, errors.New("could not parse date")
		}
		clYear = t.Year()
		clMonth = t.Month()
		clDay = t.Day()
	}

	// Construct time from parsed components
	baseTime := time.Date(clYear, clMonth, clDay, clHour, clMin, clSec, 0, clLoc)

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
	return baseTime, nil
}

// FallbackDate gets an equivalent day-of-week within fallbackWeek if currentTime is not with startTime/endTime bounds
func FallbackDate(currentTime time.Time, startTime time.Time, endTime time.Time, fallbackWeek time.Time) (time.Time, bool, error) {
	loc := currentTime.Location()
	startTime = midnight(startTime, loc)
	endTime = midnight(endTime, loc).AddDate(0, 0, 1)
	if endTime.Before(startTime) {
		return currentTime, false, errors.New("end time before start time")
	}
	if currentTime.Before(startTime) || currentTime.After(endTime) {
		// fmt.Println("using fallback: ", fallbackTime, "date:", baseTime, "bounds:", startTime, endTime)
		ft := midnight(fallbackWeek, currentTime.Location())
		for i := 0; i < 7; i++ {
			if ft.Weekday() == currentTime.Weekday() {
				currentTime = time.Date(ft.Year(), ft.Month(), ft.Day(), currentTime.Hour(), currentTime.Minute(), currentTime.Second(), 0, loc)
				return currentTime, true, nil
			}
			ft = ft.AddDate(0, 0, 1)
		}
	}
	return currentTime, false, nil
}

func midnight(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}
