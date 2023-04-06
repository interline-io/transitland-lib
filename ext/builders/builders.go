package builders

import (
	"regexp"
	"strings"

	"github.com/mmcloughlin/geohash"
)

// Support methods and types

type point struct {
	lon float64
	lat float64
}

type stopGeom struct {
	name string
	fvid int
	lon  float64
	lat  float64
}

type routeStopGeoms struct {
	agency    string
	name      string
	fvid      int
	stopGeoms map[string]*stopGeom
}

// OnestopID support functions

// filterName .
var nameTilde = regexp.MustCompile("[-:&@/]")
var nameFilter = regexp.MustCompile(`[^\pL0-9~><]`)

func filterName(name string) string {
	return strings.ToLower(nameFilter.ReplaceAllString(nameTilde.ReplaceAllString(name, "~"), ""))
}

func centroid(points []point) point {
	sumx := 0.0
	sumy := 0.0
	for _, p := range points {
		sumx += p.lon
		sumy += p.lat
	}
	return point{
		lon: sumx / float64(len(points)),
		lat: sumy / float64(len(points)),
	}
}

func pointsGeohash(points []point, minc uint, maxc uint) string {
	if len(points) == 0 {
		return ""
	}
	if minc > maxc {
		minc = maxc
	}
	c := centroid(points)
	g := geohash.EncodeWithPrecision(c.lat, c.lon, maxc)
	// t.Log("centroid:", c, "g:", g, "minc:", minc, "maxc:", maxc)
	gs := []string{}
	for _, p := range points {
		gs = append(gs, geohash.EncodeWithPrecision(p.lat, p.lon, maxc))
	}
	// t.Log("points:", gs)
	for i := maxc; i >= minc; i-- {
		check := g[0:i]
		m := map[string]bool{}
		for _, n := range geohash.Neighbors(check) {
			m[n] = true
		}
		m[check] = true
		// t.Log(i, "checking:", check, "neighbors:", m)
		allOk := true
		for _, j := range gs {
			if _, ok := m[j[0:i]]; !ok {
				allOk = false
			}
		}
		if allOk {
			// t.Log("ok:", check)
			return check
		}
	}
	return ""
}
