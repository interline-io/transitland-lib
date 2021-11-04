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

var nameTilde = "[-:&@/]"
var nameFilter = "[^[:alnum:]~><]"
var geohashFilter = "[^0123456789bcdefghjkmnpqrstuvwxyz]"

// filterName .
func filterName(name string) string {
	re1 := regexp.MustCompile(nameTilde)
	re2 := regexp.MustCompile(nameFilter)
	return strings.ToLower(re2.ReplaceAllString(re1.ReplaceAllString(name, "~"), ""))
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

func pointsGeohash(points []point) string {
	c := centroid(points)
	g := geohash.Encode(c.lat, c.lon)
	gs := []string{}
	for _, p := range points {
		gs = append(gs, geohash.Encode(p.lat, p.lon))
	}
	for i := 1; i < len(g)-1; i++ {
		r := g[0:i]
		m := map[string]int{}
		for _, n := range geohash.Neighbors(g[0:i]) {
			m[n]++
		}
		m[r]++
		b := false
		for _, j := range gs {
			if _, ok := m[j[0:i]]; !ok {
				b = true
			}
		}
		if b {
			return g[0 : i-1]
		}
	}
	return g
}
