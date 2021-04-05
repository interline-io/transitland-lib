package tlcsv

import (
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl"
)

func makeRow(header, value string) Row {
	h := strings.Split(header, ",")
	v := strings.Split(value, ",")
	hindex := makehindex(h)
	return Row{Header: h, Row: v, Hindex: hindex}
}

func makehindex(header []string) map[string]int {
	hindex := map[string]int{}
	for k, i := range header {
		hindex[i] = k
	}
	return hindex
}

func Benchmark_StopTime_Memory(b *testing.B) {
	count := 1000
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		stoptimes := make([]tl.StopTime, 0, count)
		_ = stoptimes
	}
}

// Benchmark StopTime memory usage
func Benchmark_StopTime_Memory_Read1000(b *testing.B) {
	count := 1000
	reader, err := NewReader(testutil.RelPath("test/data/bart.zip"))
	if err != nil {
		b.Error(err)
		return
	}
	b.ReportAllocs()
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		stoptimes := []tl.StopTime{}
		for st := range reader.StopTimes() {
			if len(stoptimes) >= count {
				break
			}
			stoptimes = append(stoptimes, st)
		}
	}
}

// Benchmark fast path loading
func Benchmark_loadRow_StopTime(b *testing.B) {
	row := makeRow(
		"trip_id,arrival_time,departure_time,stop_id,stop_sequence,stop_headsign,pickup_type,drop_off_type,shape_dist_traveled",
		"AAMV4,16:00:00,16:00:00,BEATTY_AIRPORT,2",
	)
	e := tl.StopTime{}
	for n := 0; n < b.N; n++ {
		loadRow(&e, row)
	}
}
func Benchmark_loadRowFast_StopTime(b *testing.B) {
	row := makeRow(
		"trip_id,arrival_time,departure_time,stop_id,stop_sequence,stop_headsign,pickup_type,drop_off_type,shape_dist_traveled",
		"AAMV4,16:00:00,16:00:00,BEATTY_AIRPORT,2",
	)
	e := tl.StopTime{}
	for n := 0; n < b.N; n++ {
		loadRowFast(&e, row)
	}
}

func Benchmark_loadRow_Shape(b *testing.B) {
	row := makeRow("shape_id,shape_pt_lat,shape_pt_lon,shape_pt_sequence,shape_dist_traveled", "a,30.0,30.0,3")
	e := tl.Shape{}
	for n := 0; n < b.N; n++ {
		loadRow(&e, row)
	}
}
func Benchmark_loadRowFast_Shape(b *testing.B) {
	row := makeRow("shape_id,shape_pt_lat,shape_pt_lon,shape_pt_sequence,shape_dist_traveled", "a,30.0,30.0,3")
	e := tl.Shape{}
	for n := 0; n < b.N; n++ {
		loadRowFast(&e, row)
	}
}

// Benchmark reflect path loading
func Benchmark_loadRow_Stop(b *testing.B) {
	row := makeRow("stop_id,stop_name,stop_desc,stop_lat,stop_lon,zone_id,stop_url", "FUR_CREEK_RES,Furnace Creek Resort (Demo),,36.425288,-117.133162,,")
	e := tl.Stop{}
	for n := 0; n < b.N; n++ {
		loadRow(&e, row)
	}
}

func Benchmark_loadRow_Calendar(b *testing.B) {
	row := makeRow("service_id,monday,tuesday,wednesday,thursday,friday,saturday,sunday,start_date,end_date", "FULLW,1,1,1,1,1,1,1,20070101,20101231")
	for n := 0; n < b.N; n++ {
		e := tl.Calendar{}
		loadRow(&e, row)
	}
}

func Benchmark_loadRow_Trip(b *testing.B) {
	row := makeRow("route_id,service_id,trip_id,trip_headsign,direction_id,block_id,shape_id", "AB,FULLW,AB1,to Bullfrog,0,1,")
	for n := 0; n < b.N; n++ {
		e := tl.Trip{}
		loadRow(&e, row)
	}
}

func Benchmark_dumpRow_StopTime(b *testing.B) {
	ent := tl.StopTime{
		TripID:            "xyz",
		StopID:            "abc",
		StopHeadsign:      tl.NewOString("hello"),
		StopSequence:      123,
		ArrivalTime:       3600,
		DepartureTime:     7200,
		ShapeDistTraveled: tl.NewOFloat(123.456),
	}
	header := strings.Split("trip_id,arrival_time,departure_time,stop_id,stop_sequence,stop_headsign,pickup_type,drop_off_type,shape_dist_traveled", ",")
	for n := 0; n < b.N; n++ {
		dumpRow(&ent, header)
	}
}
