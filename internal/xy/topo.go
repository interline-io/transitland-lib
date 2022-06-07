package xy

import "fmt"

func RemoveIndex(s []int, index int) []int {
	return append(s[:index], s[index+1:]...)
}

type Segment [2]Point

func TopoSimplify(lines [][]Point) [][]Point {
	sharedPoints := map[Point]int{}
	segs := map[Segment]bool{}
	for _, line := range lines {
		sharedPoints[line[0]] += 2
		sharedPoints[line[len(line)-1]] += 2
		for i := 1; i < len(line); i++ {
			sharedPoints[line[i]] += 1
			segs[Segment{line[i-1], line[i]}] = true
		}
	}
	fmt.Println("shared points:", sharedPoints)
	lineSegs := [][]Segment{}
	for pt, v := range sharedPoints {
		delete(sharedPoints, pt)
		if v < 2 {
			continue
		}
		fmt.Println("processing point:", pt)
		lineSeg := []Segment{}
		// get open seg
		for {
			found := false
			nseg := Segment{}
			for nseg = range segs {
				if nseg[0] == pt {
					// ok
					delete(segs, nseg)
					found = true
					break
				}
				// if nseg[1] == pt {
				// 	// ok
				// 	delete(segs, nseg)
				// 	nseg = Segment{nseg[1], nseg[0]}
				// 	found = true
				// 	break
				// }
			}
			if !found {
				fmt.Println("\t\tno seg for pt:", pt)
				lineSegs = append(lineSegs, lineSeg)
				lineSeg = nil
				break
			}
			lineSeg = append(lineSeg, nseg)
			pt = nseg[1]
			fmt.Println("\t\tfound:", found, "nseg:", nseg)
			fmt.Println("\t\tnow:", lineSeg)
		}
	}
	ret := [][]Point{}
	for _, lineSeg := range lineSegs {
		rl := []Point{lineSeg[0][0], lineSeg[0][1]}
		for i := 1; i < len(lineSeg); i++ {
			rl = append(rl, lineSeg[i][1])
		}
		ret = append(ret, rl)
	}
	fmt.Println("result:")
	for _, line := range ret {
		fmt.Println(line)
	}
	return ret
}
