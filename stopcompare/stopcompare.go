// Package stopcompare provides geometric comparison of GTFS feeds based on stop point clouds.
package stopcompare

import (
	"math"
	"sort"

	"github.com/golang/geo/s2"
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/tidwall/rtree"
)

// Relationship describes the spatial relationship between two feeds' stop sets.
type Relationship int

const (
	RelationshipUnknown     Relationship = iota
	RelationshipSame                     // A and B cover the same stops
	RelationshipSubset                   // A ⊆ B: A's stops are geographically contained in B
	RelationshipSuperset                 // A ⊇ B: B's stops are contained in A
	RelationshipOverlapping              // Partial geographic overlap
	RelationshipDisjoint                 // No meaningful geographic overlap
)

func (r Relationship) MarshalJSON() ([]byte, error) {
	return []byte(`"` + r.String() + `"`), nil
}

func (r Relationship) String() string {
	switch r {
	case RelationshipSame:
		return "same"
	case RelationshipSubset:
		return "subset"
	case RelationshipSuperset:
		return "superset"
	case RelationshipOverlapping:
		return "overlapping"
	case RelationshipDisjoint:
		return "disjoint"
	default:
		return "unknown"
	}
}

// Options controls the thresholds used for classification.
type Options struct {
	// Normalized ANND at or below which stops are "well matched". Default: 0.02
	ANNDRatioThreshold float64
	// Bounding box IoU at or above which feeds cover the "same" region. Default: 0.75
	BboxIoUThreshold float64
	// Bounding box overlap coefficient at or above which one feed is a geographic subset. Default: 0.90
	BboxOverlapThreshold float64
	// If true, only consider stops with LocationType==0 (boarding stops). Default: false
	BoardingStopsOnly bool
}

// DefaultOptions returns Options with recommended defaults.
func DefaultOptions() Options {
	return Options{
		ANNDRatioThreshold:   0.02,
		BboxIoUThreshold:     0.75,
		BboxOverlapThreshold: 0.90,
	}
}

// DirectionStats holds nearest-neighbor statistics from one feed to another.
type DirectionStats struct {
	Direction        string  `json:"direction"`
	TotalStops       int     `json:"total_stops"`
	MeanDistMeters   float64 `json:"mean_dist_meters"`
	MedianDistMeters float64 `json:"median_dist_meters"`
	P90DistMeters    float64 `json:"p90_dist_meters"`
	MaxDistMeters    float64 `json:"max_dist_meters"`
	// NormalizedANND is the mean nearest-neighbor distance divided by the bounding
	// box diagonal of the larger feed. ~0 means tight match; ~1 means far apart.
	NormalizedANND float64 `json:"normalized_annd"`
}

// BboxMetrics holds bounding-box overlap statistics.
type BboxMetrics struct {
	IoU                float64 `json:"iou"`
	OverlapCoefficient float64 `json:"overlap_coefficient"`
}

// Result is the full output of comparing two feeds.
type Result struct {
	FeedA        string          `json:"feed_a"`
	FeedB        string          `json:"feed_b"`
	StopCountA   int             `json:"stop_count_a"`
	StopCountB   int             `json:"stop_count_b"`
	AtoB         DirectionStats  `json:"a_to_b"`
	BtoA         DirectionStats  `json:"b_to_a"`
	Bbox         BboxMetrics     `json:"bbox"`
	Relationship Relationship    `json:"relationship"`
}

// stopEntry is stored in the R-tree.
type stopEntry struct {
	pt tlxy.Point
}

// pointIndex is a spatial index over a set of stop points.
type pointIndex struct {
	idx   rtree.Generic[stopEntry]
	bbox  s2.Rect
	count int
}

func newPointIndex(pts []tlxy.Point) *pointIndex {
	pi := &pointIndex{
		bbox: s2.EmptyRect(),
	}
	for _, p := range pts {
		pi.insert(p)
	}
	return pi
}

func (pi *pointIndex) insert(p tlxy.Point) {
	xy := [2]float64{p.Lon, p.Lat}
	pi.idx.Insert(xy, xy, stopEntry{pt: p})
	ll := s2.LatLngFromDegrees(p.Lat, p.Lon)
	pi.bbox = pi.bbox.AddPoint(ll)
	pi.count++
}

// nearestDist returns the distance in meters from p to the nearest point in the index.
func (pi *pointIndex) nearestDist(p tlxy.Point) float64 {
	// Start with a small search radius and expand until a candidate is found.
	approx := tlxy.NewApprox(p)
	radiusMeters := 1000.0
	for i := 0; i < 20; i++ {
		dLon := radiusMeters / approx.LonMeters()
		dLat := radiusMeters / approx.LatMeters()
		minPt := [2]float64{p.Lon - dLon, p.Lat - dLat}
		maxPt := [2]float64{p.Lon + dLon, p.Lat + dLat}
		best := -1.0
		pi.idx.Search(minPt, maxPt, func(min, max [2]float64, entry stopEntry) bool {
			d := tlxy.DistanceHaversine(p, entry.pt)
			if best < 0 || d < best {
				best = d
			}
			return true
		})
		if best >= 0 && best <= radiusMeters {
			return best
		}
		radiusMeters *= 4
	}
	return math.MaxFloat64
}

// bboxDiagonalMeters returns the length of the bounding box diagonal in meters.
func bboxDiagonalMeters(r s2.Rect) float64 {
	if r.IsEmpty() {
		return 0
	}
	lo := r.Lo()
	hi := r.Hi()
	a := tlxy.Point{Lon: lo.Lng.Degrees(), Lat: lo.Lat.Degrees()}
	b := tlxy.Point{Lon: hi.Lng.Degrees(), Lat: hi.Lat.Degrees()}
	return tlxy.DistanceHaversine(a, b)
}

// computeBboxMetrics computes IoU and overlap coefficient for two bounding boxes.
func computeBboxMetrics(rectA, rectB s2.Rect) BboxMetrics {
	if rectA.IsEmpty() || rectB.IsEmpty() {
		return BboxMetrics{}
	}
	intersect := rectA.Intersection(rectB)
	areaA := rectA.Area()
	areaB := rectB.Area()
	areaI := intersect.Area()
	areaUnion := areaA + areaB - areaI
	iou := 0.0
	if areaUnion > 0 {
		iou = areaI / areaUnion
	}
	minArea := math.Min(areaA, areaB)
	overlap := 0.0
	if minArea > 0 {
		overlap = areaI / minArea
	}
	return BboxMetrics{
		IoU:                iou,
		OverlapCoefficient: overlap,
	}
}

// computeDirectionStats computes nearest-neighbor statistics from each point in
// srcs to the nearest point in the dstIdx spatial index.
func computeDirectionStats(direction string, srcs []tlxy.Point, dstIdx *pointIndex, largerDiagonal float64) DirectionStats {
	if len(srcs) == 0 {
		return DirectionStats{Direction: direction}
	}
	dists := make([]float64, 0, len(srcs))
	for _, p := range srcs {
		d := dstIdx.nearestDist(p)
		if d < math.MaxFloat64 {
			dists = append(dists, d)
		}
	}
	if len(dists) == 0 {
		return DirectionStats{Direction: direction, TotalStops: len(srcs)}
	}
	sort.Float64s(dists)
	sum := 0.0
	for _, d := range dists {
		sum += d
	}
	mean := sum / float64(len(dists))
	median := dists[len(dists)/2]
	p90 := dists[int(math.Min(float64(len(dists)-1), math.Ceil(float64(len(dists))*0.90)))]
	maxD := dists[len(dists)-1]
	normalizedANND := 0.0
	if largerDiagonal > 0 {
		normalizedANND = mean / largerDiagonal
	}
	return DirectionStats{
		Direction:        direction,
		TotalStops:       len(srcs),
		MeanDistMeters:   mean,
		MedianDistMeters: median,
		P90DistMeters:    p90,
		MaxDistMeters:    maxD,
		NormalizedANND:   normalizedANND,
	}
}

// classifyRelationship determines the relationship from the computed metrics.
func classifyRelationship(result *Result, opts Options) Relationship {
	atob := result.AtoB.NormalizedANND
	btoa := result.BtoA.NormalizedANND
	iou := result.Bbox.IoU
	overlap := result.Bbox.OverlapCoefficient

	aWellMatched := atob <= opts.ANNDRatioThreshold
	bWellMatched := btoa <= opts.ANNDRatioThreshold

	if aWellMatched && bWellMatched && iou >= opts.BboxIoUThreshold {
		return RelationshipSame
	}
	if aWellMatched && !bWellMatched && overlap >= opts.BboxOverlapThreshold && result.StopCountA < result.StopCountB {
		return RelationshipSubset
	}
	if bWellMatched && !aWellMatched && overlap >= opts.BboxOverlapThreshold && result.StopCountA > result.StopCountB {
		return RelationshipSuperset
	}
	if iou > 0.10 {
		return RelationshipOverlapping
	}
	return RelationshipDisjoint
}

// collectStops reads stops from a reader, filtering null-island and (optionally) non-boarding stops.
func collectStops(reader adapters.Reader, boardingOnly bool) ([]tlxy.Point, error) {
	var pts []tlxy.Point
	for stop := range reader.Stops() {
		if boardingOnly && stop.LocationType.Val != 0 {
			continue
		}
		p := stop.ToPoint()
		if p.Lon == 0 && p.Lat == 0 {
			continue
		}
		pts = append(pts, p)
	}
	return pts, nil
}

// CompareReaders computes geometric comparison metrics between two open readers.
func CompareReaders(nameA string, readerA adapters.Reader, nameB string, readerB adapters.Reader, opts Options) (*Result, error) {
	stopsA, err := collectStops(readerA, opts.BoardingStopsOnly)
	if err != nil {
		return nil, err
	}
	stopsB, err := collectStops(readerB, opts.BoardingStopsOnly)
	if err != nil {
		return nil, err
	}

	idxA := newPointIndex(stopsA)
	idxB := newPointIndex(stopsB)

	bboxMetrics := computeBboxMetrics(idxA.bbox, idxB.bbox)

	diagA := bboxDiagonalMeters(idxA.bbox)
	diagB := bboxDiagonalMeters(idxB.bbox)
	largerDiag := math.Max(diagA, diagB)

	atob := computeDirectionStats(nameA+"→"+nameB, stopsA, idxB, largerDiag)
	btoa := computeDirectionStats(nameB+"→"+nameA, stopsB, idxA, largerDiag)

	result := &Result{
		FeedA:      nameA,
		FeedB:      nameB,
		StopCountA: len(stopsA),
		StopCountB: len(stopsB),
		AtoB:       atob,
		BtoA:       btoa,
		Bbox:       bboxMetrics,
	}
	result.Relationship = classifyRelationship(result, opts)
	return result, nil
}
