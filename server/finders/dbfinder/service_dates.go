package dbfinder

import (
	"sort"
	"time"
)

// DepartureLookbackDays bounds how far back a wall calendar date query reaches
// for after-midnight departures. 1 covers departures up to 48:00; raising it
// covers services that run longer, at one more service date per query.
const DepartureLookbackDays = 1

// calendarServiceDates returns the GTFS service dates that can carry a departure
// falling on any of the given wall calendar dates. A departure at time t on
// service date S falls on calendar date S + floor(t/86400), so calendar date D
// is served by service dates D-k for k in 0..lookback. Deduplicated, so a run of
// consecutive dates costs len(run)+lookback service dates, not
// len(run)*(lookback+1).
func calendarServiceDates(dates []time.Time, lookback int) []time.Time {
	seen := map[string]bool{}
	var out []time.Time
	for _, d := range dates {
		for k := 0; k <= lookback; k++ {
			s := d.AddDate(0, 0, -k)
			key := s.Format("2006-01-02")
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}
