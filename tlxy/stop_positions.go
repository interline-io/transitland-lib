package tlxy

import (
	"math"
)

// A pass this many meters farther from a stop than the nearest one still
// counts as the stop's, so a loop that runs past a stop twice keeps the
// earlier pass.
const stopLapTolerance = 15.0

// Passes farther than this many meters from a stop are not considered its.
// The first stop also takes its earliest pass within this distance of the
// nearest, since a loop's shape can end nearer its terminal than it starts.
const stopSearchRadius = 50.0

// StopPosition is where a stop sits along a shape.
type StopPosition struct {
	// Meters along the shape from its first point.
	Position float64
	// Meters from the stop to the shape at Position.
	Distance float64
	// True when the trip's shape_dist_traveled values chose the pass.
	Hinted bool
}

// StopPositions places a trip's stops along its shape, in order: each stop is
// at or after the one before, so a loop that calls at a stop twice places it
// on each pass in turn. It returns nil for a shape of fewer than two points.
//
// dists are the trip's stop_times.shape_dist_traveled values, in the feed's
// own units, or nil. They are a hint to which pass of the shape each stop is
// on: the first stop's value as its distance from the shape's start, and each
// later one by its gap from the one before. They need not match the shape's
// own distances, and values that fall, or never rise, are ignored.
func StopPositions(shape []Point, stops []Point, dists []float64) []StopPosition {
	if len(shape) < 2 {
		return nil
	}
	frame := newPlanarFrame(shape[0])
	line := newMeasuredLine(frame.line(shape))
	pts := frame.line(stops)
	// Geometry alone first, which measures the hint's scale.
	geo := placeStops(line, pts, nil, 0)
	scale, ok := hintScale(geo, dists)
	if !ok {
		return geo
	}
	return placeStops(line, pts, dists, scale)
}

// ShapeDistances is the distance along a shape at each of its points, in
// meters, in the measure StopPositions places stops by.
func ShapeDistances(shape []Point) []float64 {
	if len(shape) == 0 {
		return nil
	}
	return newMeasuredLine(newPlanarFrame(shape[0]).line(shape)).cum
}

// placeStops places each stop after the one before. With dists, each stop
// takes its pass nearest where dists put it, and falls back to geometry alone
// where no pass is near there.
func placeStops(line measuredLine, pts []planarPoint, dists []float64, scale float64) []StopPosition {
	ret := make([]StopPosition, len(pts))
	for i, p := range pts {
		minPos, tol := 0.0, stopSearchRadius
		if i > 0 {
			minPos, tol = ret[i-1].Position, stopLapTolerance
		}
		if dists != nil {
			gap := scale * dists[i]
			if i > 0 {
				gap = scale * (dists[i] - dists[i-1])
			}
			if c, ok := line.nearestPass(p, minPos, gap); ok {
				ret[i] = StopPosition{Position: c.pos, Distance: c.dist, Hinted: true}
				continue
			}
		}
		pos, dist := line.lapGuard(p, minPos, tol)
		ret[i] = StopPosition{Position: pos, Distance: dist}
	}
	return ret
}

// hintScale is meters per unit of dists: the span geometry alone measured
// from the first stop to the last, over the same span in dists. Only the two
// end stops count, so a stop on the wrong pass in between can't skew it.
func hintScale(geo []StopPosition, dists []float64) (float64, bool) {
	n := len(dists)
	if n != len(geo) || n < 2 {
		return 0, false
	}
	for i := 1; i < n; i++ {
		if d := dists[i] - dists[i-1]; math.IsNaN(d) || d < 0 {
			return 0, false
		}
	}
	span := dists[n-1] - dists[0]
	if span <= 0 {
		return 0, false
	}
	scale := (geo[n-1].Position - geo[0].Position) / span
	return scale, scale > 0
}

// planarFrame maps lon/lat to meters east and north of an origin. Over the
// extent of one shape it is close enough, and cheap enough to scan every
// segment for every stop.
type planarFrame struct {
	origin     Point
	lonM, latM float64
}

type planarPoint struct{ x, y float64 }

func newPlanarFrame(origin Point) planarFrame {
	ap := NewApprox(origin)
	return planarFrame{origin: origin, lonM: ap.LonMeters(), latM: ap.LatMeters()}
}

func (f planarFrame) line(pts []Point) []planarPoint {
	out := make([]planarPoint, len(pts))
	for i, p := range pts {
		out[i] = planarPoint{(p.Lon - f.origin.Lon) * f.lonM, (p.Lat - f.origin.Lat) * f.latM}
	}
	return out
}

func planarDist(a, b planarPoint) float64 {
	return math.Hypot(a.x-b.x, a.y-b.y)
}

// measuredLine is a planar line with the meters from its start at each point.
type measuredLine struct {
	pts []planarPoint
	cum []float64
}

func newMeasuredLine(pts []planarPoint) measuredLine {
	cum := make([]float64, len(pts))
	for i := 1; i < len(pts); i++ {
		cum[i] = cum[i-1] + planarDist(pts[i-1], pts[i])
	}
	return measuredLine{pts: pts, cum: cum}
}

// measuredPass is the nearest point of one pass of a line to a stop.
type measuredPass struct {
	pos, dist float64
}

// passes returns each pass of the line by a point at or after minPos, in order
// along the line. A pass is a local minimum of distance from the point: a
// straight road has one, a loop one per lap.
func (l measuredLine) passes(p planarPoint, minPos float64) []measuredPass {
	type seg struct {
		measuredPass
		ok bool
	}
	segs := make([]seg, len(l.pts)-1)
	for i := 1; i < len(l.pts); i++ {
		if l.cum[i] < minPos {
			continue
		}
		pos, dist := l.projectSegment(i, p, minPos)
		segs[i-1] = seg{measuredPass{pos, dist}, true}
	}
	var ret []measuredPass
	for i, s := range segs {
		if !s.ok {
			continue
		}
		prev, next := math.Inf(1), math.Inf(1)
		if i > 0 && segs[i-1].ok {
			prev = segs[i-1].dist
		}
		if i+1 < len(segs) && segs[i+1].ok {
			next = segs[i+1].dist
		}
		if s.dist <= prev && s.dist <= next {
			ret = append(ret, s.measuredPass)
		}
	}
	return ret
}

// lapGuard is the earliest pass at or after minPos whose distance is within
// tol of the nearest pass's.
func (l measuredLine) lapGuard(p planarPoint, minPos, tol float64) (float64, float64) {
	cands := l.passes(p, minPos)
	if len(cands) == 0 {
		return minPos, math.Inf(1)
	}
	nearest := cands[0].dist
	for _, c := range cands[1:] {
		nearest = math.Min(nearest, c.dist)
	}
	for _, c := range cands {
		if c.dist <= nearest+tol {
			return c.pos, c.dist
		}
	}
	// Unreachable: the nearest pass is itself within tol.
	return cands[0].pos, cands[0].dist
}

// nearestPass is the pass at or after minPos, within the search radius of p,
// nearest to minPos+gap; none when every such pass is farther from there than
// half the gap, or the search radius if that is larger.
func (l measuredLine) nearestPass(p planarPoint, minPos, gap float64) (measuredPass, bool) {
	expected := minPos + gap
	slack := math.Max(stopSearchRadius, gap/2)
	best, found := measuredPass{}, false
	for _, c := range l.passes(p, minPos) {
		off := math.Abs(c.pos - expected)
		if c.dist > stopSearchRadius || off > slack {
			continue
		}
		if !found || off < math.Abs(best.pos-expected) {
			best, found = c, true
		}
	}
	return best, found
}

// projectSegment projects p onto segment i, clamped to positions at or after
// minPos.
func (l measuredLine) projectSegment(i int, p planarPoint, minPos float64) (float64, float64) {
	a, b := l.pts[i-1], l.pts[i]
	segLen := l.cum[i] - l.cum[i-1]
	t := 0.0
	if segLen > 0 {
		t = ((p.x-a.x)*(b.x-a.x) + (p.y-a.y)*(b.y-a.y)) / (segLen * segLen)
		t = math.Max(0, math.Min(1, t))
	}
	pos := l.cum[i-1] + t*segLen
	if pos < minPos {
		// The foot lies before minPos: clamp to minPos within this segment.
		t = (minPos - l.cum[i-1]) / segLen
		pos = minPos
	}
	q := planarPoint{a.x + t*(b.x-a.x), a.y + t*(b.y-a.y)}
	return pos, planarDist(q, p)
}
