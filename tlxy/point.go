package tlxy

import "fmt"

type Point struct {
	Lon float64
	Lat float64
}

func (p *Point) String() string {
	return fmt.Sprintf("[%0.5f,%0.5f]", p.Lon, p.Lat)
}
