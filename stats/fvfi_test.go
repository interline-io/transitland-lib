package stats

import (
	"testing"

	"github.com/interline-io/transitland-lib/adapters/tlcsv"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tt"
)

func TestNewFeedVersionFileInfosFromReader(t *testing.T) {
	tcs := []struct {
		name         string
		url          string
		expectResult []dmfr.FeedVersionFileInfo
	}{
		{
			"example",
			testutil.ExampleZip.URL,
			[]dmfr.FeedVersionFileInfo{
				{
					Name:    "agency.txt",
					Size:    114,
					Rows:    1,
					Header:  "agency_id,agency_name,agency_url,agency_timezone",
					CSVLike: true,
					SHA1:    "278b80dde686714b5c394a2c26e63b44d4373b0a",
					ValuesUnique: tt.Counts{
						"agency_id":       1,
						"agency_name":     1,
						"agency_timezone": 1,
						"agency_url":      1},
					ValuesCount: tt.Counts{
						"agency_id":       1,
						"agency_name":     1,
						"agency_timezone": 1,
						"agency_url":      1}},
				{
					Name:    "calendar.txt",
					Size:    161,
					Rows:    2,
					Header:  "service_id,monday,tuesday,wednesday,thursday,friday,saturday,sunday,start_date,end_date",
					CSVLike: true,
					SHA1:    "c5854474f69cfeb7c30051412c5a788b38cbf5b5",
					ValuesUnique: tt.Counts{
						"end_date":   1,
						"friday":     2,
						"monday":     2,
						"saturday":   1,
						"service_id": 2,
						"start_date": 1,
						"sunday":     1,
						"thursday":   2,
						"tuesday":    2,
						"wednesday":  2},
					ValuesCount: tt.Counts{
						"end_date":   2,
						"friday":     2,
						"monday":     2,
						"saturday":   2,
						"service_id": 2,
						"start_date": 2,
						"sunday":     2,
						"thursday":   2,
						"tuesday":    2,
						"wednesday":  2}},
				{
					Name:    "calendar_dates.txt",
					Size:    66,
					Rows:    2,
					Header:  "service_id,date,exception_type",
					CSVLike: true,
					SHA1:    "f36297a1a47f767050f22932257ba78cd0e6fe4c",
					ValuesUnique: tt.Counts{
						"date":           1,
						"exception_type": 2,
						"service_id":     2},
					ValuesCount: tt.Counts{
						"date":           2,
						"exception_type": 2,
						"service_id":     2}},
				{
					Name:    "fare_attributes.txt",
					Size:    103,
					Rows:    2,
					Header:  "fare_id,price,currency_type,payment_method,transfers,transfer_duration",
					CSVLike: true,
					SHA1:    "59082b5bc479fe3e1dad184bd4aa8db40896f7b8",
					ValuesUnique: tt.Counts{
						"currency_type":  1,
						"fare_id":        2,
						"payment_method": 1,
						"price":          2,
						"transfers":      1},
					ValuesCount: tt.Counts{
						"currency_type":  2,
						"fare_id":        2,
						"payment_method": 2,
						"price":          2,
						"transfers":      2}},
				{
					Name:    "fare_rules.txt",
					Size:    91,
					Rows:    4,
					Header:  "fare_id,route_id,origin_id,destination_id,contains_id",
					CSVLike: true,
					SHA1:    "e156a2e5e27955526de05dd2d738ededed4d3ca7",
					ValuesUnique: tt.Counts{
						"fare_id":  2,
						"route_id": 4},
					ValuesCount: tt.Counts{
						"fare_id":  4,
						"route_id": 4}},
				{
					Name:    "feed_info.txt",
					Size:    145,
					Rows:    1,
					Header:  "feed_publisher_name,feed_publisher_url,feed_lang,feed_start_date,feed_end_date,feed_version,feed_id",
					CSVLike: true,
					SHA1:    "d815c8ff03e369053275038b480cfd4fa2ef96ad",
					ValuesUnique: tt.Counts{
						"feed_id":             1,
						"feed_lang":           1,
						"feed_publisher_name": 1,
						"feed_publisher_url":  1,
						"feed_version":        1},
					ValuesCount: tt.Counts{
						"feed_id":             1,
						"feed_lang":           1,
						"feed_publisher_name": 1,
						"feed_publisher_url":  1,
						"feed_version":        1}},
				{
					Name:    "frequencies.txt",
					Size:    346,
					Rows:    11,
					Header:  "trip_id,start_time,end_time,headway_secs",
					CSVLike: true,
					SHA1:    "ee7d2bb97df08775806ccf2a9436e20cb240886c",
					ValuesUnique: tt.Counts{
						"end_time":     5,
						"headway_secs": 2,
						"start_time":   5,
						"trip_id":      3},
					ValuesCount: tt.Counts{
						"end_time":     11,
						"headway_secs": 11,
						"start_time":   11,
						"trip_id":      11}},
				{
					Name:    "malformed.txt",
					Size:    249,
					Rows:    7,
					Header:  "rowname,this,file,checks,for,parse,errors",
					CSVLike: true,
					SHA1:    "b4daf22e7f4dd40eece1901bebdcb3a3eaf5d44b",
					ValuesUnique: tt.Counts{
						"checks":  1,
						"file":    3,
						"for":     1,
						"parse":   1,
						"rowname": 7,
						"this":    5},
					ValuesCount: tt.Counts{
						"checks":  2,
						"file":    4,
						"for":     1,
						"parse":   1,
						"rowname": 7,
						"this":    7}},
				{
					Name:    "routes.txt",
					Size:    311,
					Rows:    5,
					Header:  "route_id,agency_id,route_short_name,route_long_name,route_desc,route_type,route_url,route_color,route_text_color",
					CSVLike: true,
					SHA1:    "e40c7b96a3c50d86e87cfbbe82f451d2a0b0448a",
					ValuesUnique: tt.Counts{
						"agency_id":        1,
						"route_id":         5,
						"route_long_name":  5,
						"route_short_name": 5,
						"route_type":       1},
					ValuesCount: tt.Counts{
						"agency_id":        5,
						"route_id":         5,
						"route_long_name":  5,
						"route_short_name": 5,
						"route_type":       5}},
				{
					Name:    "shapes.txt",
					Size:    198,
					Rows:    9,
					Header:  "shape_id,shape_pt_lat,shape_pt_lon,shape_pt_sequence,shape_dist_traveled",
					CSVLike: true,
					SHA1:    "3cd7d8b9dd1e37b6f44d0f164be81d345fcfaa64",
					ValuesUnique: tt.Counts{
						"shape_id":          3,
						"shape_pt_lat":      9,
						"shape_pt_lon":      9,
						"shape_pt_sequence": 4},
					ValuesCount: tt.Counts{
						"shape_id":          9,
						"shape_pt_lat":      9,
						"shape_pt_lon":      9,
						"shape_pt_sequence": 9}},
				{
					Name:    "stop_times.txt",
					Size:    1120,
					Rows:    28,
					Header:  "trip_id,arrival_time,departure_time,stop_id,stop_sequence,stop_headsign,pickup_type,drop_off_type,shape_dist_traveled",
					CSVLike: true,
					SHA1:    "aa1e207366ec809d6c3bef40035ff941ddf94d75",
					ValuesUnique: tt.Counts{
						"arrival_time":   25,
						"departure_time": 25,
						"stop_id":        9,
						"stop_sequence":  5,
						"trip_id":        11},
					ValuesCount: tt.Counts{
						"arrival_time":   28,
						"departure_time": 28,
						"stop_id":        28,
						"stop_sequence":  28,
						"trip_id":        28}},
				{
					Name:    "stops.txt",
					Size:    597,
					Rows:    9,
					Header:  "stop_id,stop_name,stop_desc,stop_lat,stop_lon,zone_id,stop_url",
					CSVLike: true,
					SHA1:    "1ad23d6f5a7b05646bf79d1791d626cf23653d48",
					ValuesUnique: tt.Counts{
						"stop_id":   9,
						"stop_lat":  9,
						"stop_lon":  9,
						"stop_name": 9},
					ValuesCount: tt.Counts{
						"stop_id":   9,
						"stop_lat":  9,
						"stop_lon":  9,
						"stop_name": 9}},
				{
					Name:    "trips.txt",
					Size:    411,
					Rows:    11,
					Header:  "route_id,service_id,trip_id,trip_headsign,direction_id,block_id,shape_id",
					CSVLike: true,
					SHA1:    "f99d40ad90977120aa78184a7474571ea475058a",
					ValuesUnique: tt.Counts{
						"block_id":      2,
						"direction_id":  2,
						"route_id":      5,
						"service_id":    2,
						"trip_headsign": 5,
						"trip_id":       11},
					ValuesCount: tt.Counts{
						"block_id":      4,
						"direction_id":  10,
						"route_id":      11,
						"service_id":    11,
						"trip_headsign": 9,
						"trip_id":       11}},
			},
		},
		{
			"bart",
			testutil.ExampleFeedBART.URL,
			[]dmfr.FeedVersionFileInfo{
				{
					Name:    "agency.txt",
					Size:    134,
					Rows:    1,
					Header:  "agency_id,agency_name,agency_url,agency_timezone,agency_lang",
					CSVLike: true,
					SHA1:    "e37ed8048b32e944602d08172112e870ffef1d35",
					ValuesUnique: tt.Counts{
						"agency_id":       1,
						"agency_lang":     1,
						"agency_name":     1,
						"agency_timezone": 1,
						"agency_url":      1},
					ValuesCount: tt.Counts{
						"agency_id":       1,
						"agency_lang":     1,
						"agency_name":     1,
						"agency_timezone": 1,
						"agency_url":      1}},
				{
					Name:    "calendar.txt",
					Size:    201,
					Rows:    3,
					Header:  "service_id,monday,tuesday,wednesday,thursday,friday,saturday,sunday,start_date,end_date",
					CSVLike: true,
					SHA1:    "cf1a295028fa3ae58d513e099772098db24d9306",
					ValuesUnique: tt.Counts{
						"end_date":   1,
						"friday":     2,
						"monday":     2,
						"saturday":   2,
						"service_id": 3,
						"start_date": 1,
						"sunday":     2,
						"thursday":   2,
						"tuesday":    2,
						"wednesday":  2},
					ValuesCount: tt.Counts{
						"end_date":   3,
						"friday":     3,
						"monday":     3,
						"saturday":   3,
						"service_id": 3,
						"start_date": 3,
						"sunday":     3,
						"thursday":   3,
						"tuesday":    3,
						"wednesday":  3}},
				{
					Name:    "calendar_dates.txt",
					Size:    230,
					Rows:    12,
					Header:  "service_id,date,exception_type",
					CSVLike: true,
					SHA1:    "22c10081c9f8da19ff78c878e198648fbe06aa9e",
					ValuesUnique: tt.Counts{
						"date":           6,
						"exception_type": 2,
						"service_id":     2},
					ValuesCount: tt.Counts{
						"date":           12,
						"exception_type": 12,
						"service_id":     12}},
				{
					Name:    "fare_attributes.txt",
					Size:    3124,
					Rows:    170,
					Header:  "fare_id,price,currency_type,payment_method,transfers,transfer_duration",
					CSVLike: true,
					SHA1:    "23b108a41818d0d60021a7f60a58478ba406fc37",
					ValuesUnique: tt.Counts{
						"currency_type":  1,
						"fare_id":        170,
						"payment_method": 1,
						"price":          170},
					ValuesCount: tt.Counts{
						"currency_type":  170,
						"fare_id":        170,
						"payment_method": 170,
						"price":          170}},
				{
					Name:    "fare_rules.txt",
					Size:    38199,
					Rows:    2304,
					Header:  "fare_id,route_id,origin_id,destination_id,contains_id",
					CSVLike: true,
					SHA1:    "4a098a27198b69a25dcb6b77bd66edd55942f546",
					ValuesUnique: tt.Counts{
						"destination_id": 48,
						"fare_id":        170,
						"origin_id":      48},
					ValuesCount: tt.Counts{
						"destination_id": 2304,
						"fare_id":        2304,
						"origin_id":      2304}},
				{
					Name:    "feed_info.txt",
					Size:    161,
					Rows:    1,
					Header:  "feed_publisher_name,feed_publisher_url,feed_lang,feed_start_date,feed_end_date,feed_version",
					CSVLike: true,
					SHA1:    "acc955965f492cdcc79fff87b34fc8acdb6d619e",
					ValuesUnique: tt.Counts{
						"feed_end_date":       1,
						"feed_lang":           1,
						"feed_publisher_name": 1,
						"feed_publisher_url":  1,
						"feed_start_date":     1,
						"feed_version":        1},
					ValuesCount: tt.Counts{
						"feed_end_date":       1,
						"feed_lang":           1,
						"feed_publisher_name": 1,
						"feed_publisher_url":  1,
						"feed_start_date":     1,
						"feed_version":        1}},
				{
					Name:         "frequencies.txt",
					Size:         42,
					Rows:         0,
					Header:       "trip_id,start_time,end_time,headway_secs",
					CSVLike:      true,
					SHA1:         "3fb52d9114b02a68c9e21a83b077a2dc86b52c06",
					ValuesUnique: tt.Counts{},
					ValuesCount:  tt.Counts{}},
				{
					Name:    "routes.txt",
					Size:    742,
					Rows:    6,
					Header:  "route_id,agency_id,route_short_name,route_long_name,route_desc,route_type,route_url,route_color,route_text_color",
					CSVLike: true,
					SHA1:    "dfbae34dcaaa70bda308b9fc687ea0a289a7b1c4",
					ValuesUnique: tt.Counts{
						"agency_id":       1,
						"route_color":     6,
						"route_id":        6,
						"route_long_name": 6,
						"route_type":      1,
						"route_url":       6},
					ValuesCount: tt.Counts{
						"agency_id":       6,
						"route_color":     6,
						"route_id":        6,
						"route_long_name": 6,
						"route_type":      6,
						"route_url":       6}},
				{
					Name:    "shapes.txt",
					Size:    874894,
					Rows:    25074,
					Header:  "shape_id,shape_pt_lat,shape_pt_lon,shape_pt_sequence",
					CSVLike: true,
					SHA1:    "5d568ecc4c15e4d3dcbe1c98fa489ff5a1fc836c",
					ValuesUnique: tt.Counts{
						"shape_id":          12,
						"shape_pt_lat":      6768,
						"shape_pt_lon":      8382,
						"shape_pt_sequence": 5232},
					ValuesCount: tt.Counts{
						"shape_id":          25074,
						"shape_pt_lat":      25074,
						"shape_pt_lon":      25074,
						"shape_pt_sequence": 25074}},
				{
					Name:    "stop_times.txt",
					Size:    1896334,
					Rows:    33167,
					Header:  "trip_id,arrival_time,departure_time,stop_id,stop_sequence,stop_headsign,pickup_type,drop_off_type,shape_dist_traveled,timepoint",
					CSVLike: true,
					SHA1:    "609765f2314a6399992181c338a106103d96dbee",
					ValuesUnique: tt.Counts{
						"arrival_time":   1287,
						"departure_time": 1287,
						"stop_headsign":  16,
						"stop_id":        50,
						"stop_sequence":  28,
						"timepoint":      1,
						"trip_id":        2525},
					ValuesCount: tt.Counts{
						"arrival_time":   33167,
						"departure_time": 33167,
						"stop_headsign":  33167,
						"stop_id":        33167,
						"stop_sequence":  33167,
						"timepoint":      33167,
						"trip_id":        33167}},
				{
					Name:    "stops.txt",
					Size:    4626,
					Rows:    50,
					Header:  "stop_id,stop_name,stop_desc,stop_lat,stop_lon,zone_id,stop_url,location_type,parent_station,stop_timezone,wheelchair_boarding",
					CSVLike: true,
					SHA1:    "e97b70f0396d82c040f03ed640a9ecf34178bb0c",
					ValuesUnique: tt.Counts{
						"location_type":       1,
						"stop_id":             50,
						"stop_lat":            48,
						"stop_lon":            48,
						"stop_name":           48,
						"stop_url":            48,
						"wheelchair_boarding": 1,
						"zone_id":             48},
					ValuesCount: tt.Counts{
						"location_type":       50,
						"stop_id":             50,
						"stop_lat":            50,
						"stop_lon":            50,
						"stop_name":           50,
						"stop_url":            50,
						"wheelchair_boarding": 50,
						"zone_id":             50}},
				{
					Name:    "transfers.txt",
					Size:    182,
					Rows:    8,
					Header:  "from_stop_id,to_stop_id,transfer_type,min_transfer_time",
					CSVLike: true,
					SHA1:    "a752eb00c0a4f4f3cae7fd1977a820dcfc6c334d",
					ValuesUnique: tt.Counts{
						"from_stop_id":      8,
						"min_transfer_time": 1,
						"to_stop_id":        8,
						"transfer_type":     3},
					ValuesCount: tt.Counts{
						"from_stop_id":      8,
						"min_transfer_time": 1,
						"to_stop_id":        8,
						"transfer_type":     8}},
				{
					Name:    "trips.txt",
					Size:    126693,
					Rows:    2525,
					Header:  "route_id,service_id,trip_id,trip_headsign,direction_id,block_id,shape_id,wheelchair_accessible,bikes_allowed",
					CSVLike: true,
					SHA1:    "5020b32d02dbfdfe5fdd10f422290eaf1095887b",
					ValuesUnique: tt.Counts{
						"bikes_allowed":         1,
						"direction_id":          2,
						"route_id":              6,
						"service_id":            3,
						"shape_id":              12,
						"trip_headsign":         16,
						"trip_id":               2525,
						"wheelchair_accessible": 1},
					ValuesCount: tt.Counts{
						"bikes_allowed":         2525,
						"direction_id":          2525,
						"route_id":              2525,
						"service_id":            2525,
						"shape_id":              2525,
						"trip_headsign":         2525,
						"trip_id":               2525,
						"wheelchair_accessible": 2525}},
			},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			reader, err := tlcsv.NewReader(tc.url)
			if err != nil {
				t.Fatal(err)
			}
			results, err := NewFeedVersionFileInfosFromReader(reader)
			if err != nil {
				t.Error(err)
			}
			// for _, a := range results {
			// 	type testFvfi struct {
			// 		Name         string
			// 		Size         int64
			// 		Rows         int64
			// 		Header       string
			// 		CSVLike      bool
			// 		SHA1         string
			// 		ValuesUnique tt.Counts
			// 		ValuesCount  tt.Counts
			// 	}
			// 	x := testFvfi{
			// 		Name:         a.Name,
			// 		Size:         a.Size,
			// 		Rows:         a.Rows,
			// 		Header:       a.Header,
			// 		CSVLike:      a.CSVLike,
			// 		SHA1:         a.SHA1,
			// 		ValuesUnique: a.ValuesUnique,
			// 		ValuesCount:  a.ValuesCount,
			// 	}
			// 	fmt.Println(strings.ReplaceAll(fmt.Sprintf("%#v,", x), "dmfr.testFvfi", ""))
			// }
			for _, check := range tc.expectResult {
				t.Run(check.Name, func(t *testing.T) {
					match := false
					for _, a := range results {
						if check.Name == a.Name &&
							check.Size == a.Size &&
							check.Rows == a.Rows &&
							check.Header == a.Header &&
							check.CSVLike == a.CSVLike &&
							check.SHA1 == a.SHA1 &&
							compareCounts(check.ValuesCount, a.ValuesCount) &&
							compareCounts(check.ValuesUnique, a.ValuesUnique) {
							match = true
						}
					}
					if !match {
						t.Errorf("no match for %#v\n", check)
					}
				})
			}
		})
	}
}

func compareCounts(a tt.Counts, b tt.Counts) bool {
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}
