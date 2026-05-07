package stopcompare

import (
	"math"
	"testing"

	"github.com/golang/geo/s2"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/stretchr/testify/assert"
)

// --- pointIndex tests ---

func TestPointIndex_NearestDist(t *testing.T) {
	pts := []tlxy.Point{
		{Lon: -122.0, Lat: 37.0},
		{Lon: -121.9, Lat: 37.1},
		{Lon: -122.1, Lat: 36.9},
	}
	idx := newPointIndex(pts)

	// Querying exactly an indexed point should return ~0
	d := idx.nearestDist(pts[0])
	assert.InDelta(t, 0.0, d, 1.0, "exact match should be ~0m")

	// A point near but not equal to an indexed point
	near := tlxy.Point{Lon: -122.0001, Lat: 37.0001}
	d2 := idx.nearestDist(near)
	assert.Greater(t, d2, 0.0)
	assert.Less(t, d2, 50.0, "nearby point should match within 50m")
}

func TestPointIndex_BboxGrows(t *testing.T) {
	pts := []tlxy.Point{
		{Lon: -74.0, Lat: 40.7},
		{Lon: -73.9, Lat: 40.8},
	}
	idx := newPointIndex(pts)
	assert.False(t, idx.bbox.IsEmpty())
	assert.Equal(t, 2, idx.count)
}

// --- bboxMetrics tests ---

func TestComputeBboxMetrics_IdenticalRects(t *testing.T) {
	r := s2.RectFromLatLng(s2.LatLngFromDegrees(37.0, -122.0))
	r = r.AddPoint(s2.LatLngFromDegrees(38.0, -121.0))
	m := computeBboxMetrics(r, r)
	assert.InDelta(t, 1.0, m.IoU, 1e-9, "identical rects should have IoU=1")
	assert.InDelta(t, 1.0, m.OverlapCoefficient, 1e-9, "identical rects should have overlap=1")
}

func TestComputeBboxMetrics_DisjointRects(t *testing.T) {
	a := s2.RectFromLatLng(s2.LatLngFromDegrees(37.0, -122.0))
	a = a.AddPoint(s2.LatLngFromDegrees(38.0, -121.0))
	b := s2.RectFromLatLng(s2.LatLngFromDegrees(40.0, -74.0))
	b = b.AddPoint(s2.LatLngFromDegrees(41.0, -73.0))
	m := computeBboxMetrics(a, b)
	assert.InDelta(t, 0.0, m.IoU, 1e-9, "non-overlapping rects should have IoU=0")
	assert.InDelta(t, 0.0, m.OverlapCoefficient, 1e-9)
}

func TestComputeBboxMetrics_OneContained(t *testing.T) {
	outer := s2.RectFromLatLng(s2.LatLngFromDegrees(37.0, -122.0))
	outer = outer.AddPoint(s2.LatLngFromDegrees(39.0, -120.0))
	inner := s2.RectFromLatLng(s2.LatLngFromDegrees(37.5, -121.5))
	inner = inner.AddPoint(s2.LatLngFromDegrees(38.5, -120.5))
	m := computeBboxMetrics(outer, inner)
	// overlap coefficient should be near 1 (inner is fully within outer)
	assert.Greater(t, m.OverlapCoefficient, 0.99, "inner fully inside outer → overlap≈1")
	// IoU < 1 since union > intersection
	assert.Less(t, m.IoU, 1.0)
}

// --- computeDirectionStats tests ---

func TestComputeDirectionStats_ZeroDistance(t *testing.T) {
	pts := []tlxy.Point{
		{Lon: -122.0, Lat: 37.0},
		{Lon: -121.9, Lat: 37.1},
	}
	idx := newPointIndex(pts)
	diag := bboxDiagonalMeters(idx.bbox)
	stats := computeDirectionStats("A→B", pts, idx, diag)
	assert.InDelta(t, 0.0, stats.MeanDistMeters, 1.0, "same stops → mean~0")
	assert.InDelta(t, 0.0, stats.NormalizedANND, 0.001)
}

func TestComputeDirectionStats_LargeDistance(t *testing.T) {
	srcs := []tlxy.Point{
		{Lon: -74.0, Lat: 40.7}, // NYC
	}
	dsts := []tlxy.Point{
		{Lon: -122.4, Lat: 37.8}, // SF
	}
	idx := newPointIndex(dsts)
	// Build a bbox spanning both cities for normalization
	allPts := append(srcs, dsts...)
	combinedIdx := newPointIndex(allPts)
	diag := bboxDiagonalMeters(combinedIdx.bbox)
	stats := computeDirectionStats("A→B", srcs, idx, diag)
	// Cross-country distance is ~4000km; diag is ~4500km → ANND ~0.9
	assert.Greater(t, stats.NormalizedANND, 0.5, "NYC→SF ANND should be large")
}

// --- classifyRelationship tests ---

func makeResult(atobANND, btoaANND, iou, overlap float64, countA, countB int) *Result {
	return &Result{
		StopCountA: countA,
		StopCountB: countB,
		AtoB:       DirectionStats{NormalizedANND: atobANND},
		BtoA:       DirectionStats{NormalizedANND: btoaANND},
		Bbox:       BboxMetrics{IoU: iou, OverlapCoefficient: overlap},
	}
}

func TestClassifyRelationship(t *testing.T) {
	opts := DefaultOptions()
	cases := []struct {
		name     string
		result   *Result
		expected Relationship
	}{
		{
			"same",
			makeResult(0.01, 0.01, 0.90, 0.95, 100, 100),
			RelationshipSame,
		},
		{
			"subset (A ⊆ B)",
			makeResult(0.01, 0.05, 0.60, 0.95, 50, 200),
			RelationshipSubset,
		},
		{
			"superset (A ⊇ B)",
			makeResult(0.05, 0.01, 0.60, 0.95, 200, 50),
			RelationshipSuperset,
		},
		{
			"overlapping",
			makeResult(0.10, 0.10, 0.50, 0.60, 100, 100),
			RelationshipOverlapping,
		},
		{
			"disjoint",
			makeResult(0.80, 0.80, 0.05, 0.05, 100, 100),
			RelationshipDisjoint,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyRelationship(tc.result, opts)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// --- pipeline tests using internal functions ---

func TestCompareReaders_Same(t *testing.T) {
	pts := makeSFStops()
	result := runCompare("A", pts, "B", pts)
	assert.Equal(t, RelationshipSame, result.Relationship)
}

func TestCompareReaders_Disjoint(t *testing.T) {
	sfPts := makeSFStops()
	nycPts := makeNYCStops()
	result := runCompare("SF", sfPts, "NYC", nycPts)
	assert.Equal(t, RelationshipDisjoint, result.Relationship)
}

func TestCompareReaders_Subset(t *testing.T) {
	// A is a small cluster; B contains A plus many more stops spread over a larger area
	aStops := []tlxy.Point{
		{Lon: -122.41, Lat: 37.78},
		{Lon: -122.42, Lat: 37.79},
		{Lon: -122.40, Lat: 37.77},
	}
	bStops := make([]tlxy.Point, len(aStops))
	copy(bStops, aStops)
	for i := 0; i < 50; i++ {
		lon := -122.0 + float64(i)*0.05
		lat := 37.0 + float64(i)*0.02
		bStops = append(bStops, tlxy.Point{Lon: lon, Lat: lat})
	}
	result := runCompare("A", aStops, "B", bStops)
	assert.Equal(t, RelationshipSubset, result.Relationship)
}

// runCompare runs the metric pipeline over two point slices and classifies them.
func runCompare(nameA string, ptsA []tlxy.Point, nameB string, ptsB []tlxy.Point) *Result {
	opts := DefaultOptions()
	idxA := newPointIndex(ptsA)
	idxB := newPointIndex(ptsB)
	bm := computeBboxMetrics(idxA.bbox, idxB.bbox)
	diagA := bboxDiagonalMeters(idxA.bbox)
	diagB := bboxDiagonalMeters(idxB.bbox)
	largerDiag := math.Max(diagA, diagB)
	atob := computeDirectionStats(nameA+"→"+nameB, ptsA, idxB, largerDiag)
	btoa := computeDirectionStats(nameB+"→"+nameA, ptsB, idxA, largerDiag)
	result := &Result{
		FeedA:      nameA,
		FeedB:      nameB,
		StopCountA: len(ptsA),
		StopCountB: len(ptsB),
		AtoB:       atob,
		BtoA:       btoa,
		Bbox:       bm,
	}
	result.Relationship = classifyRelationship(result, opts)
	return result
}

// --- fixture helpers ---

func makeSFStops() []tlxy.Point {
	return []tlxy.Point{
		{Lon: -122.4194, Lat: 37.7749},
		{Lon: -122.4089, Lat: 37.7833},
		{Lon: -122.4314, Lat: 37.7680},
		{Lon: -122.3960, Lat: 37.7910},
		{Lon: -122.4500, Lat: 37.8010},
	}
}

func makeNYCStops() []tlxy.Point {
	return []tlxy.Point{
		{Lon: -74.0060, Lat: 40.7128},
		{Lon: -73.9857, Lat: 40.7580},
		{Lon: -74.0445, Lat: 40.6892},
		{Lon: -73.9442, Lat: 40.6501},
		{Lon: -73.9969, Lat: 40.7282},
	}
}
