package tlxy

import (
	"math"
	"sort"
)

// A pass this many meters farther from a stop than the nearest one still
// counts as the stop's, so a loop that runs past a stop twice keeps the
// earlier pass.
const stopLapTolerance = 15.0

// Two passes by a stop count as two only when the shape moves this many
// meters farther from the stop between them, rather than turning a corner.
const stopPassRise = 30.0

// The hint takes only a pass within this many meters of its stop. A trip's
// first stop takes its earliest pass within this distance of its nearest,
// since a loop's shape can start farther from its terminal than it ends.
const stopSearchRadius = 50.0

// A stop farther than this many meters from its shape has no position.
const stopMaxDistance = 500.0

// StopPosition is where a stop sits along a shape.
type StopPosition struct {
	// Meters along the shape from its first point.
	Position float64
	// Meters from the stop to the shape at Position.
	Distance float64
	// True when the trip's shape_dist_traveled values chose the pass.
	Hinted bool
	// False for a stop too far from the shape, or without finite coordinates;
	// the other fields are then zero.
	Placed bool
}

// StopPositions places a trip's stops along its shape, in order: each stop is
// at or after the one before, so a loop that calls at a stop twice places it
// on each pass in turn. It returns nil for a shape of fewer than two points,
// without length, or with a point without finite coordinates.
//
// dists are the trip's stop_times.shape_dist_traveled values, in the feed's
// own units, or nil. They hint at each stop's pass, by its gap in dists from
// the stop before, or for the first stop by its value from the shape's start.
// They need not match the shape's own distances, and values that fall, or
// never rise, are ignored. Geometry alone places each stop the hint can't,
// but not past the stretch where the hint finds the next stop.
func StopPositions(shape []Point, stops []Point, dists []float64) []StopPosition {
	line, ok := newMeasuredLine(shape)
	if !ok {
		return nil
	}
	n := len(stops)
	scale, hinted := 0.0, false
	if len(dists) == n && rising(dists) {
		scale, hinted = line.hintScale(stops[0], stops[n-1], dists[n-1]-dists[0])
	}
	ret := make([]StopPosition, n)
	// The last stop placed is at minPos, with prevDist its value in dists, or
	// before any is placed, the shape's start.
	minPos, prevDist := 0.0, 0.0
	for i, p := range stops {
		if !finitePoint(p) {
			continue
		}
		c, fromHint := measuredPass{}, false
		if hinted {
			c, fromHint = line.nearestPass(p, minPos, minPos+scale*(dists[i]-prevDist))
		}
		if !fromHint {
			maxPos := line.length()
			// Keep this stop off the stretch past where the hint finds the next
			// one, so a pass far along can't drag the stops after it.
			if hinted && i+1 < n && finitePoint(stops[i+1]) {
				expected := minPos + scale*(dists[i+1]-prevDist)
				if _, ok := line.nearestPass(stops[i+1], minPos, expected); ok {
					_, maxPos = hintWindow(minPos, expected)
				}
			}
			c = line.geometryPass(p, i, n, minPos, maxPos)
			if c.dist > stopMaxDistance {
				continue
			}
		}
		ret[i] = StopPosition{Position: c.pos, Distance: c.dist, Hinted: fromHint, Placed: true}
		minPos = c.pos
		if hinted {
			prevDist = dists[i]
		}
	}
	return ret
}

// ShapeDistances is the distance along a shape at each of its points, in
// meters: LengthHaversine of the shape up to each point. It returns nil for a
// shape with a point without finite coordinates.
func ShapeDistances(shape []Point) []float64 {
	var ret []float64
	total := 0.0
	for i, p := range shape {
		if !finitePoint(p) {
			return nil
		}
		if i > 0 {
			total += DistanceHaversine(shape[i-1], p)
		}
		ret = append(ret, total)
	}
	return ret
}

// hintScale is meters per unit of dists: the meters from the first stop's
// earliest pass to the last stop's latest, each within stopMaxDistance of its
// stop, over span. Only the end stops count, each placed along the whole
// shape on its own, so no stop between them can skew the scale.
func (l measuredLine) hintScale(first, last Point, span float64) (float64, bool) {
	if !finitePoint(first) || !finitePoint(last) {
		return 0, false
	}
	start, end := 0.0, 0.0
	foundStart, foundEnd := false, false
	for _, c := range l.passes(first, 0, l.length()) {
		if c.dist <= stopMaxDistance {
			start, foundStart = c.pos, true
			break
		}
	}
	for _, c := range l.passes(last, 0, l.length()) {
		if c.dist <= stopMaxDistance {
			end, foundEnd = c.pos, true
		}
	}
	scale := (end - start) / span
	return scale, foundStart && foundEnd && scale > 0
}

// hintWindow is the stretch of the shape where the hint looks for a stop it
// expects at expected, after the last stop placed at minPos: within the
// search radius of expected, or half the gap from minPos if that is larger.
func hintWindow(minPos, expected float64) (float64, float64) {
	slack := math.Max(stopSearchRadius, (expected-minPos)/2)
	return expected - slack, expected + slack
}

// geometryPass is the pass that geometry alone gives stop i of n, between
// minPos and maxPos. A trip's last stop takes its latest pass within the lap
// tolerance of its nearest, so on a shape that loops at its terminal the stop
// sits at the loop's end.
func (l measuredLine) geometryPass(p Point, i, n int, minPos, maxPos float64) measuredPass {
	cands := l.passes(p, minPos, maxPos)
	switch i {
	case 0:
		return nearOnes(cands, stopSearchRadius)[0]
	case n - 1:
		near := nearOnes(cands, stopLapTolerance)
		return near[len(near)-1]
	}
	return nearOnes(cands, stopLapTolerance)[0]
}

// nearestPass is p's pass at or after minPos nearest expected, of those within
// the search radius of p and the hint's window. Only passes within the lap
// tolerance of the nearest such pass to p compete, so the tip of a spike in
// the shape can't win.
func (l measuredLine) nearestPass(p Point, minPos, expected float64) (measuredPass, bool) {
	lo, hi := hintWindow(minPos, expected)
	var near []measuredPass
	for _, c := range l.passes(p, minPos, l.length()) {
		if c.dist <= stopSearchRadius && c.pos >= lo && c.pos <= hi {
			near = append(near, c)
		}
	}
	if len(near) == 0 {
		return measuredPass{}, false
	}
	near = nearOnes(near, stopLapTolerance)
	best := near[0]
	for _, c := range near[1:] {
		if math.Abs(c.pos-expected) < math.Abs(best.pos-expected) {
			best = c
		}
	}
	return best, true
}

// nearOnes is cands within tol of the nearest one's distance, in order.
func nearOnes(cands []measuredPass, tol float64) []measuredPass {
	nearest := math.Inf(1)
	for _, c := range cands {
		nearest = math.Min(nearest, c.dist)
	}
	var ret []measuredPass
	for _, c := range cands {
		if c.dist <= nearest+tol {
			ret = append(ret, c)
		}
	}
	return ret
}

// rising reports whether dists are finite and never fall, and the last is
// greater than the first.
func rising(dists []float64) bool {
	for i, d := range dists {
		if math.IsNaN(d) || math.IsInf(d, 0) || (i > 0 && d < dists[i-1]) {
			return false
		}
	}
	return len(dists) > 1 && dists[len(dists)-1] > dists[0]
}

func finitePoint(p Point) bool {
	return !math.IsNaN(p.Lon) && !math.IsInf(p.Lon, 0) && !math.IsNaN(p.Lat) && !math.IsInf(p.Lat, 0)
}

// measuredLine is a shape with the meters along it at each point, and each
// segment in a planar frame of its own.
type measuredLine struct {
	pts  []Point
	cum  []float64
	segs []segment
}

// segment is the segment from one point of a measuredLine to the next, in
// meters east and north of its start, at the scale of its middle latitude.
type segment struct {
	lonM   float64 // meters per degree of longitude
	dx, dy float64
}

// newMeasuredLine measures shape, or returns false for a shape of fewer than
// two points, without length, or with a point without finite coordinates.
func newMeasuredLine(shape []Point) (measuredLine, bool) {
	cum := ShapeDistances(shape)
	if len(cum) < 2 || cum[len(cum)-1] == 0 {
		return measuredLine{}, false
	}
	segs := make([]segment, len(shape)-1)
	for i := range segs {
		a, b := shape[i], shape[i+1]
		lonM := latMeter * math.Cos(deg2rad((a.Lat+b.Lat)/2))
		segs[i] = segment{lonM: lonM, dx: wrapLon(b.Lon-a.Lon) * lonM, dy: (b.Lat - a.Lat) * latMeter}
	}
	return measuredLine{pts: shape, cum: cum, segs: segs}, true
}

func (l measuredLine) length() float64 {
	return l.cum[len(l.cum)-1]
}

// wrapLon wraps a difference in longitude into [-180, 180], so a segment
// across the antimeridian stays short.
func wrapLon(d float64) float64 {
	return d - 360*math.Round(d/360)
}

// measuredPass is the nearest point of one pass of a line to a stop.
type measuredPass struct {
	pos, dist float64
}

// passes returns each pass of the line by p between positions lo and hi, in
// order along the line. A pass is a local minimum of the line's distance from
// p, kept apart from the next by a rise of stopPassRise: a straight road has
// one, a loop one per lap, and a road out and back one each way. A minimum at
// lo or hi is always a pass of its own, so the ends of the range stay
// choices.
func (l measuredLine) passes(p Point, lo, hi float64) []measuredPass {
	// The range ends with the line, so one that starts at the line's end is a
	// single point.
	hi = math.Min(hi, l.length())
	// The pass at the range's start, if any, and the passes after it.
	var start, ret []measuredPass
	// The farthest the line gets from p since the last pass.
	peak := 0.0
	add := func(c measuredPass) {
		n := len(ret)
		switch {
		case n == 0 || peak-math.Max(ret[n-1].dist, c.dist) >= stopPassRise:
			ret = append(ret, c)
		case c.dist < ret[n-1].dist:
			// The same pass, nearer here.
			ret[n-1] = c
		default:
			return
		}
		peak = c.dist
	}
	first, prevAt, last := true, 0, measuredPass{}
	// Segments that end before lo can't reach the range.
	for i := sort.SearchFloat64s(l.cum[1:], lo); i < len(l.segs); i++ {
		s := l.segs[i]
		segLen := l.cum[i+1] - l.cum[i]
		// Skip repeated points, segments outside [lo, hi], and segments that
		// only touch it, unless it is a single point.
		overlap := math.Min(l.cum[i+1], hi) - math.Max(l.cum[i], lo)
		if segLen == 0 || overlap < 0 || (overlap == 0 && hi > lo) {
			continue
		}
		a := l.pts[i]
		px, py := wrapLon(p.Lon-a.Lon)*s.lonM, (p.Lat-a.Lat)*latMeter
		dist := func(t float64) float64 { return math.Hypot(px-t*s.dx, py-t*s.dy) }
		// p's foot on the part of the segment within [lo, hi]; at is -1 or 1
		// where the foot clamps to that part's start or end.
		t0, t1 := math.Max(0, (lo-l.cum[i])/segLen), math.Min(1, (hi-l.cum[i])/segLen)
		t, at := (px*s.dx+py*s.dy)/(s.dx*s.dx+s.dy*s.dy), 0
		if t <= t0 {
			t, at = t0, -1
		} else if t >= t1 {
			t, at = t1, 1
		}
		c := measuredPass{pos: l.cum[i] + t*segLen, dist: dist(t)}
		peak = math.Max(peak, dist(t0))
		// A foot inside the segment is a pass, and so is one at its start where
		// the range starts or the segment before ends at the same point.
		switch {
		case at < 0 && first:
			start = append(start, c)
		case at == 0 || (at < 0 && prevAt > 0):
			add(c)
		}
		peak = math.Max(peak, dist(t1))
		first, prevAt, last = false, at, c
	}
	// So is a foot at the end of the range.
	if prevAt > 0 {
		ret = append(ret, last)
	}
	return append(start, ret...)
}
