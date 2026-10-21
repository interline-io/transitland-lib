package model

import (
	"context"
	"sort"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
)

// Now is the time a request runs at: its configured clock's, or the system's.
func Now(ctx context.Context) time.Time {
	if clock := ForContext(ctx).Clock; clock != nil {
		return clock.Now()
	}
	return time.Now()
}

// FeedServiceWindow is the stretch of a feed version's regular service that date
// filters resolve dates against, and the week that stands in for dates outside
// it.
type FeedServiceWindow struct {
	StartDate    time.Time
	EndDate      time.Time
	FallbackWeek time.Time
	Location     *time.Location
}

// ServiceWindow is a feed version's service window as of a request's time.
type ServiceWindow struct {
	FeedServiceWindow
	NowLocal time.Time
}

// At returns the window as of now, in the feed version's timezone.
func (w FeedServiceWindow) At(now time.Time) *ServiceWindow {
	return &ServiceWindow{FeedServiceWindow: w, NowLocal: now.In(w.Location)}
}

// NewFeedServiceWindow chooses a feed version's service window, in loc, its
// default timezone, from the stats its fetch computed. The window is its
// feed_info dates when it has both, otherwise the weeks of regular service its
// service levels show; see serviceLevelWindow.
func NewFeedServiceWindow(fvsw dmfr.FeedVersionServiceWindow, levels []dmfr.FeedVersionServiceLevel, fetchedAt time.Time, loc *time.Location) FeedServiceWindow {
	start, end, fallbackWeek := serviceLevelWindow(levels, fetchedAt)
	if fvsw.FeedStartDate.Valid && fvsw.FeedEndDate.Valid && !fvsw.FeedStartDate.Val.IsZero() && !fvsw.FeedEndDate.Val.IsZero() {
		start, end = fvsw.FeedStartDate.Val, fvsw.FeedEndDate.Val
	}
	return FeedServiceWindow{
		StartDate:    tzTruncate(start, loc).Val,
		EndDate:      tzTruncate(end, loc).Val,
		FallbackWeek: tzTruncate(fallbackWeek, loc).Val,
		Location:     loc,
	}
}

// minServiceRatio is the share of the busiest week's service a week needs to
// count as regular service.
const minServiceRatio = 0.75

func hasFullService(sl dmfr.FeedVersionServiceLevel) bool {
	return sl.Monday > 0 && sl.Tuesday > 0 && sl.Wednesday > 0 && sl.Thursday > 0 && sl.Friday > 0 && sl.Saturday > 0 && sl.Sunday > 0
}

// serviceLevelWindow finds the weeks of regular service in a feed version's
// weekly service levels, and the week to stand in for dates outside them.
//
// It starts from the week the feed version was fetched in, or the busiest week
// when that one is missing or has less than minServiceRatio of the busiest
// week's service, and extends in both directions while weeks keep that ratio.
func serviceLevelWindow(levels []dmfr.FeedVersionServiceLevel, fetchedAt time.Time) (startDate, endDate, fallbackWeek time.Time) {
	if len(levels) == 0 {
		return
	}
	levels = append([]dmfr.FeedVersionServiceLevel(nil), levels...)
	sort.SliceStable(levels, func(i, j int) bool { return levels[i].StartDate.Val.Before(levels[j].StartDate.Val) })

	// Get the week which includes fetched_at date, and the highest service week
	highestIdx := 0
	highestService := -1
	fetchedWeek := -1
	for i, ent := range levels {
		sd := ent.StartDate.Val
		ed := ent.EndDate.Val
		if (sd.Before(fetchedAt) || sd.Equal(fetchedAt)) && (ed.After(fetchedAt) || ed.Equal(fetchedAt)) {
			fetchedWeek = i
		}
		if ent.Total() > highestService {
			highestIdx = i
			highestService = ent.Total()
		}
	}
	if fetchedWeek < 0 {
		fetchedWeek = highestIdx
	}
	// If the fetched week has bad service, use highest week
	if float64(levels[fetchedWeek].Total())/float64(highestService) < minServiceRatio {
		fetchedWeek = highestIdx
	}

	// Expand window in both directions from chosen week
	startDate = levels[fetchedWeek].StartDate.Val
	endDate = levels[fetchedWeek].EndDate.Val
	for i := fetchedWeek; i < len(levels); i++ {
		ent := levels[i]
		if float64(ent.Total())/float64(highestService) < minServiceRatio {
			break
		}
		if ent.StartDate.Val.Before(startDate) {
			startDate = ent.StartDate.Val
		}
		endDate = ent.EndDate.Val
	}
	for i := fetchedWeek - 1; i > 0; i-- {
		ent := levels[i]
		if float64(ent.Total())/float64(highestService) < minServiceRatio {
			break
		}
		if ent.EndDate.Val.After(endDate) {
			endDate = ent.EndDate.Val
		}
		startDate = ent.StartDate.Val
	}
	return startDate, endDate, bestWeek(levels, startDate, endDate)
}

// bestWeek returns the start of the busiest week in the window with service on
// every day, or failing that the busiest week in it. Levels are in date order.
func bestWeek(levels []dmfr.FeedVersionServiceLevel, startDate, endDate time.Time) time.Time {
	best := levels[0].StartDate.Val
	bestService := levels[0].Total()
	hasFullServiceWeek := false
	isWeekInWindow := func(ent dmfr.FeedVersionServiceLevel) bool {
		sd := ent.StartDate.Val
		ed := ent.EndDate.Val
		return (sd.Before(endDate) || sd.Equal(endDate)) && (ed.After(startDate) || ed.Equal(startDate))
	}
	// First pass: look for weeks with service on all days
	for _, ent := range levels {
		if isWeekInWindow(ent) && hasFullService(ent) && ent.Total() > bestService {
			bestService = ent.Total()
			best = ent.StartDate.Val
			hasFullServiceWeek = true
		}
	}
	// Second pass: if no full-service week found, fall back to any week with highest service
	if !hasFullServiceWeek {
		for _, ent := range levels {
			if isWeekInWindow(ent) && ent.Total() > bestService {
				bestService = ent.Total()
				best = ent.StartDate.Val
			}
		}
	}
	return best
}
