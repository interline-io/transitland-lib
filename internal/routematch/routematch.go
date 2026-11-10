// Package routematch compares the trips of two routes that may be copies of the
// same service published by different feeds: it pairs their trips, says how the
// two sets of trips relate, and pairs the stops that matched trips use at the
// same place.
//
// It knows nothing about the database; the finder supplies each trip's start
// and first stop, and each stop pattern's stops.
package routematch

import (
	"sort"
	"time"

	"github.com/interline-io/transitland-lib/tlxy"
)

// Options holds the thresholds; DefaultOptions gives the values the API
// documents.
type Options struct {
	// Two trips can match only when they start within this many seconds.
	StartToleranceSeconds int
	// ...and their first stops are within this many meters, which keeps a
	// trip from matching one that leaves the far end of the line at the same
	// time.
	FirstStopMaxMeters float64
	// A route's trips "run in" another route when at least this share of them
	// are matched.
	RunInFraction float64
	// Stops of two matched trips pair only within this many meters.
	CorrespondenceMaxMeters float64
}

// DefaultOptions returns the documented thresholds.
func DefaultOptions() Options {
	return Options{
		StartToleranceSeconds:   60,
		FirstStopMaxMeters:      500,
		RunInFraction:           0.9,
		CorrespondenceMaxMeters: 300,
	}
}

// TripStart is one run of a trip on one service date. A frequency-based trip
// gives one TripStart per run.
type TripStart struct {
	// Service date of the run.
	Date time.Time
	// Seconds since midnight of the service date; may be 24:00:00 or later.
	Start int
	// Where the run's first stop is.
	FirstStop tlxy.Point
	// Identifies the run's stop pattern, for StopCorrespondences.
	PatternKey int
}

// abs is the run's start as seconds since the Unix epoch day of its service
// date, so a run written as 24:15 on one date and one written as 00:15 on the
// next compare as the same time.
func (t TripStart) abs() int64 {
	y, m, d := t.Date.Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Unix() / 86400
	return day*86400 + int64(t.Start)
}

// Pair is a matched trip of each route.
type Pair struct {
	A TripStart
	B TripStart
}

// MatchTrips pairs trips of a with trips of b, each trip with at most one
// other: for each trip of a, in start order, the unmatched trip of b that
// starts closest in time within the tolerance and whose first stop is close
// enough.
func MatchTrips(a, b []TripStart, opt Options) []Pair {
	as := sortedByStart(a)
	bs := sortedByStart(b)
	used := make([]bool, len(bs))
	tol := int64(opt.StartToleranceSeconds)
	var pairs []Pair
	lo := 0
	for _, ta := range as {
		at := ta.abs()
		for lo < len(bs) && bs[lo].abs() < at-tol {
			lo++
		}
		best := -1
		var bestDiff int64
		for k := lo; k < len(bs) && bs[k].abs() <= at+tol; k++ {
			if used[k] {
				continue
			}
			if tlxy.DistanceHaversine(ta.FirstStop, bs[k].FirstStop) > opt.FirstStopMaxMeters {
				continue
			}
			diff := bs[k].abs() - at
			if diff < 0 {
				diff = -diff
			}
			if best < 0 || diff < bestDiff {
				best, bestDiff = k, diff
			}
		}
		if best >= 0 {
			used[best] = true
			pairs = append(pairs, Pair{A: ta, B: bs[best]})
		}
	}
	return pairs
}

func sortedByStart(ts []TripStart) []TripStart {
	ret := append([]TripStart(nil), ts...)
	sort.SliceStable(ret, func(i, j int) bool { return ret[i].abs() < ret[j].abs() })
	return ret
}

// Relationship says how two routes' trips compare.
type Relationship string

// The relationships, from the point of view of the first route.
const (
	SameTrips    Relationship = "SAME_TRIPS"
	Subset       Relationship = "SUBSET"
	Superset     Relationship = "SUPERSET"
	Partial      Relationship = "PARTIAL"
	Disjoint     Relationship = "DISJOINT"
	NoTrips      Relationship = "NO_TRIPS"
	OtherNoTrips Relationship = "OTHER_NO_TRIPS"
)

// Relate says how a route with `trips` trips compares with one with
// `otherTrips`, given `matched` one-to-one pairs between them.
func Relate(trips, otherTrips, matched int, opt Options) Relationship {
	switch {
	case trips == 0:
		return NoTrips
	case otherTrips == 0:
		return OtherNoTrips
	case matched == 0:
		return Disjoint
	}
	runsIn := float64(matched)/float64(trips) >= opt.RunInFraction
	otherRunsIn := float64(matched)/float64(otherTrips) >= opt.RunInFraction
	switch {
	case runsIn && otherRunsIn:
		return SameTrips
	case runsIn:
		return Subset
	case otherRunsIn:
		return Superset
	}
	return Partial
}

// PatternStop is a stop in a stop pattern, in order.
type PatternStop struct {
	StopID int
	Point  tlxy.Point
}

// Correspondence pairs a stop of the first route with the stop of the second
// that matched trips use at the same place.
type Correspondence struct {
	StopID       int
	OtherStopID  int
	MatchedTrips int
	Distance     float64
}

// StopCorrespondences pairs the stops that matched trips use at the same place.
// Pairs that follow the same two stop patterns are aligned once and counted for
// each pair. patternsA and patternsB give each pattern's stops by PatternKey.
func StopCorrespondences(pairs []Pair, patternsA, patternsB map[int][]PatternStop, opt Options) []Correspondence {
	type patternPair struct{ a, b int }
	counts := map[patternPair]int{}
	for _, p := range pairs {
		counts[patternPair{p.A.PatternKey, p.B.PatternKey}]++
	}
	type stopPair struct{ a, b int }
	byStops := map[stopPair]*Correspondence{}
	for pp, n := range counts {
		// A loop can pair the same two stops twice; each trip counts once.
		seen := map[stopPair]bool{}
		for _, al := range alignPatterns(patternsA[pp.a], patternsB[pp.b], opt.CorrespondenceMaxMeters) {
			key := stopPair{al.StopID, al.OtherStopID}
			if seen[key] {
				continue
			}
			seen[key] = true
			c, ok := byStops[key]
			if !ok {
				c = &Correspondence{StopID: al.StopID, OtherStopID: al.OtherStopID, Distance: al.Distance}
				byStops[key] = c
			}
			c.MatchedTrips += n
		}
	}
	ret := make([]Correspondence, 0, len(byStops))
	for _, c := range byStops {
		ret = append(ret, *c)
	}
	sort.Slice(ret, func(i, j int) bool {
		if ret[i].MatchedTrips != ret[j].MatchedTrips {
			return ret[i].MatchedTrips > ret[j].MatchedTrips
		}
		if ret[i].StopID != ret[j].StopID {
			return ret[i].StopID < ret[j].StopID
		}
		return ret[i].OtherStopID < ret[j].OtherStopID
	})
	return ret
}

// alignPatterns pairs the stops of two patterns in order, as many pairs within
// maxMeters as possible and, among alignments with the most pairs, the closest.
// Each pair scores between 0.5 and 1, more the closer the stops, so one more
// pair always outweighs closeness. A stop only one pattern serves pairs with
// nothing, and the alignment never crosses itself, so a loop that passes a stop
// twice pairs each visit with its own counterpart.
func alignPatterns(a, b []PatternStop, maxMeters float64) []Correspondence {
	n, m := len(a), len(b)
	if n == 0 || m == 0 || maxMeters <= 0 {
		return nil
	}
	dist := make([][]float64, n)
	score := make([][]float64, n+1)
	for i := range score {
		score[i] = make([]float64, m+1)
	}
	for i := 0; i < n; i++ {
		dist[i] = make([]float64, m)
		for j := 0; j < m; j++ {
			dist[i][j] = tlxy.DistanceHaversine(a[i].Point, b[j].Point)
		}
	}
	pairScore := func(i, j int) (float64, bool) {
		d := dist[i][j]
		if d > maxMeters {
			return 0, false
		}
		return 1 - d/(2*maxMeters), true
	}
	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			best := max(score[i-1][j], score[i][j-1])
			if s, ok := pairScore(i-1, j-1); ok && score[i-1][j-1]+s > best {
				best = score[i-1][j-1] + s
			}
			score[i][j] = best
		}
	}
	// Walk back from the end, taking a pair wherever it made the score.
	var ret []Correspondence
	for i, j := n, m; i > 0 && j > 0; {
		if s, ok := pairScore(i-1, j-1); ok && score[i][j] == score[i-1][j-1]+s {
			ret = append(ret, Correspondence{StopID: a[i-1].StopID, OtherStopID: b[j-1].StopID, Distance: dist[i-1][j-1]})
			i, j = i-1, j-1
		} else if score[i-1][j] >= score[i][j-1] {
			i--
		} else {
			j--
		}
	}
	for l, r := 0, len(ret)-1; l < r; l, r = l+1, r-1 {
		ret[l], ret[r] = ret[r], ret[l]
	}
	return ret
}
