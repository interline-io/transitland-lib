package copier

import (
	"sort"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/internal/log"
)

// showVisited prints info about a fileInfo map
func showVisited(marker marker, s0 string, s1 string, s2 string) bool {
	fis := map[string]fileInfo{}
	switch v := marker.(type) {
	case *allMarker:
		fis = v.fileInfos
	case *visitedMarker:
		fis = v.fileInfos
	default:
		return true
	}
	// Sort
	keys := []string{}
	for k := range fis {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	// How many visited?
	log.Info("%s:", s0)
	totalVisited := 0
	totalMarked := 0
	for _, k := range keys {
		v := fis[k]
		visited := 0
		for _, v := range v.Visited {
			visited += v
		}
		totalVisited += visited
		marked := 0
		for _, v := range v.Marked {
			marked += v
		}
		totalMarked += marked
		vlen := len(v.Visited)
		mlen := len(v.Marked)
		if len(s2) == 0 {
			if vlen != visited {
				log.Info("\t%s: %s %d (%d groups)", k, s1, visited, vlen)
			} else {
				log.Info("\t%s: %s %d", k, s1, visited)
			}
		} else {
			if visited == 0 && marked == 0 {
				log.Info("\t%s: 0 %s, 0 %s", k, s1, s2)
			} else if (vlen != visited || mlen != marked) && mlen > 0 {
				markpct := (float64(marked) / float64(visited)) * 100.0
				lenpct := (float64(mlen) / float64(vlen)) * 100.0
				log.Info("\t%s: %s %d (%d groups), %s %d (%d groups), %0.2f%% (%0.2f%% groups)", k, s1, marked, mlen, s2, visited, vlen, markpct, lenpct)
			} else {
				seenpct := (float64(marked) / float64(visited)) * 100.0
				log.Info("\t%s: %s %d, %s %d, %0.2f%%", k, s1, marked, s2, visited, seenpct)
			}
		}
	}
	if len(s2) == 0 {
		log.Info("\ttotal: %s %d", s1, totalVisited)
	} else {
		log.Info("\ttotal: %s %d, %s %d", s1, totalMarked, s2, totalVisited)
	}
	return totalMarked == totalVisited
}

// createExpectMarker marks all entities from Reader, with the marked entities of m2 as Visited
func createExpectMarker(reader gotransit.Reader, m2 marker) allMarker {
	m1 := newAllMarker()
	m1.VisitAndMark(reader)
	// Get Marked entities from m2
	fi2 := map[string]fileInfo{}
	switch v := m2.(type) {
	case *allMarker:
		fi2 = v.fileInfos
	case *visitedMarker:
		fi2 = v.fileInfos
	}
	// Get all filenames
	fns := map[string]bool{}
	for fn := range m1.fileInfos {
		fns[fn] = true
	}
	for fn := range fi2 {
		fns[fn] = true
	}
	// Merge
	for fn := range fns {
		fi, ok := m1.fileInfos[fn]
		if !ok {
			fi = newFileInfo()
			m1.fileInfos[fn] = fi
		}
		fi.Visited = map[string]int{}
		if expect, ok := fi2[fn]; ok {
			fi.Visited = expect.Marked
		}
		m1.fileInfos[fn] = fi
	}
	return m1
}
