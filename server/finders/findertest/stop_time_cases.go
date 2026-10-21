package findertest

// Cases duplicated from server/gql/stop_resolver_st_test.go.

func stopResolverStopTimesDatesTestcases() []testcase {
	q := `query($stop_id:String!,$sd:Date,$date:Date,$ed:Boolean,$start:Seconds,$end:Seconds){ stops(where:{stop_id:$stop_id}) { stop_times(where:{service_date:$sd, date:$date, start:$start, end:$end, use_service_window:$ed}) {date service_date arrival_time}}}`
	testcases := []testcase{
		// Service date parameter
		{
			name:         "service_date in range",
			query:        q,
			vars:         hw{"stop_id": "MCAR_S", "sd": "2018-05-29", "start": "15:00:00", "end": "16:00:00", "ed": true},
			selector:     "stops.0.stop_times.0.service_date",
			selectExpect: []string{"2018-05-29"}, // expect input date
		},
		// Answered from the fallback week, which the exact-dates case below shows
		// has the only service, but reported under the requested date.
		{
			name:         "service_date after range",
			query:        q,
			vars:         hw{"stop_id": "MCAR_S", "sd": "2030-05-28", "start": "15:00:00", "end": "16:00:00", "ed": true},
			selector:     "stops.0.stop_times.0.service_date",
			selectExpect: []string{"2030-05-28"},
		},
		{
			name:         "service_date before range, friday",
			query:        q,
			vars:         hw{"stop_id": "MCAR_S", "sd": "2010-05-28", "start": "15:00:00", "end": "16:00:00", "ed": true},
			selector:     "stops.0.stop_times.0.service_date",
			selectExpect: []string{"2010-05-28"},
		},
		{
			name:         "service_date after range, exact dates",
			query:        q,
			vars:         hw{"stop_id": "MCAR_S", "sd": "2030-05-28", "start": "15:00:00", "end": "16:00:00", "ed": false},
			selector:     "stops.0.stop_times.#.service_date",
			selectExpect: []string{}, // exect no results
		},
		// Date parameter
		{
			name:         "date 2018-05-29 3pm-4pm",
			query:        q,
			vars:         hw{"stop_id": "MCAR_S", "date": "2018-05-29", "start": "15:00:00", "end": "16:00:00"},
			selector:     "stops.0.stop_times.#.arrival_time",
			selectExpect: []string{"15:01:00", "15:09:00", "15:09:00", "15:16:00", "15:24:00", "15:24:00", "15:31:00", "15:39:00", "15:39:00", "15:46:00", "15:54:00", "15:54:00"},
		},
		// Previous day
		{
			name:         "date 2018-05-28 12:15am - 2018-05-19 5am",
			query:        q,
			vars:         hw{"stop_id": "MCAR_S", "date": "2018-05-29", "start": "00:15:00", "end": "5:00:00"},
			selector:     "stops.0.stop_times.#.arrival_time",
			selectExpect: []string{"24:15:00", "24:15:00", "24:47:00", "24:47:00", "04:31:00", "04:39:00", "04:39:00", "04:46:00", "04:54:00", "04:54:00"},
		},
		// Next day
		{
			name:         "date 2018-05-28 10:00pm - 2018-05-29 2am",
			query:        q,
			vars:         hw{"stop_id": "MCAR_S", "date": "2018-05-28", "start": "22:00:00", "end": "26:00:00"},
			selector:     "stops.0.stop_times.#.arrival_time",
			selectExpect: []string{"22:15:00", "22:15:00", "22:35:00", "22:35:00", "22:55:00", "22:55:00", "23:15:00", "23:15:00", "23:35:00", "23:35:00", "23:55:00", "23:55:00", "24:15:00", "24:15:00", "24:47:00", "24:47:00"},
		},
		{
			name:         "date 2018-05-28 10:00pm - 2018-05-29 5am",
			query:        q,
			vars:         hw{"stop_id": "MCAR_S", "date": "2018-05-28", "start": "22:00:00", "end": "29:00:00"},
			selector:     "stops.0.stop_times.#.arrival_time",
			selectExpect: []string{"22:15:00", "22:15:00", "22:35:00", "22:35:00", "22:55:00", "22:55:00", "23:15:00", "23:15:00", "23:35:00", "23:35:00", "23:55:00", "23:55:00", "24:15:00", "24:15:00", "24:47:00", "24:47:00", "04:31:00", "04:39:00", "04:39:00", "04:46:00", "04:54:00", "04:54:00"},
		},
		// Check date, service date
		{
			name:         "date 2018-05-28 10:00pm - 2018-05-29 2am check date",
			query:        q,
			vars:         hw{"stop_id": "MCAR_S", "date": "2018-05-28", "start": "22:00:00", "end": "26:00:00"},
			selector:     "stops.0.stop_times.#.date",
			selectExpect: []string{"2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-29", "2018-05-29", "2018-05-29", "2018-05-29"},
		},
		{
			name:         "date 2018-05-28 10:00pm - 2018-05-29 2am check service date",
			query:        q,
			vars:         hw{"stop_id": "MCAR_S", "date": "2018-05-28", "start": "22:00:00", "end": "26:00:00"},
			selector:     "stops.0.stop_times.#.service_date",
			selectExpect: []string{"2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28", "2018-05-28"},
		},
	}
	return testcases
}

func stopResolverStopTimesDateWindowsTestcases() []testcase {
	bartWeekdayTimes := []string{"15:01:00", "15:09:00", "15:09:00", "15:16:00", "15:24:00", "15:24:00", "15:31:00", "15:39:00", "15:39:00", "15:46:00", "15:54:00", "15:54:00"}
	bartWeekendTimes := []string{"15:15:00", "15:15:00", "15:35:00", "15:35:00", "15:55:00", "15:55:00"}
	q := `query($stop_id:String!,$sd:Date!,$ed:Boolean){ stops(where:{stop_id:$stop_id}) { stop_times(where:{service_date:$sd, start_time:54000, end_time:57600, use_service_window:$ed}) {arrival_time}}}`
	testcases := []testcase{
		{
			name:         "service date in range",
			query:        q,
			vars:         hw{"stop_id": "MCAR_S", "sd": "2018-05-29", "ed": true},
			selector:     "stops.0.stop_times.#.arrival_time",
			selectExpect: bartWeekdayTimes,
		},
		{
			name:         "service date after range",
			query:        q,
			vars:         hw{"stop_id": "MCAR_S", "sd": "2030-05-28", "ed": true},
			selector:     "stops.0.stop_times.#.arrival_time",
			selectExpect: bartWeekdayTimes,
		},
		{
			name:         "service date after range, exact dates",
			query:        q,
			vars:         hw{"stop_id": "MCAR_S", "sd": "2030-05-28", "ed": false},
			selector:     "stops.0.stop_times.#.arrival_time",
			selectExpect: []string{},
		},
		{
			name:         "service date after range, sunday",
			query:        q,
			vars:         hw{"stop_id": "MCAR_S", "sd": "2030-05-26", "ed": true},
			selector:     "stops.0.stop_times.#.arrival_time",
			selectExpect: bartWeekendTimes,
		},
		{
			name:         "service date before range, tuesday",
			query:        q,
			vars:         hw{"stop_id": "MCAR_S", "sd": "2010-05-28", "ed": true},
			selector:     "stops.0.stop_times.#.arrival_time",
			selectExpect: bartWeekdayTimes,
		},
		{
			name:         "fv without feed_info, in window, monday",
			query:        q,
			vars:         hw{"stop_id": "70011", "sd": "2019-02-11", "ed": true},
			selector:     "stops.0.stop_times.#.arrival_time",
			selectExpect: []string{"15:48:00", "15:50:00"},
		},
		{
			name:         "fv without feed_info, before window, friday",
			query:        q,
			vars:         hw{"stop_id": "70011", "sd": "2010-05-28", "ed": true},
			selector:     "stops.0.stop_times.#.arrival_time",
			selectExpect: []string{"15:48:00", "15:50:00"},
		},
		{
			name:         "fv without feed_info, after window, tuesday",
			query:        q,
			vars:         hw{"stop_id": "70011", "sd": "2030-05-28", "ed": true},
			selector:     "stops.0.stop_times.#.arrival_time",
			selectExpect: []string{"15:48:00", "15:50:00"},
		},
		{
			name:         "fv without feed_info, after window, tuesday, exact date only",
			query:        q,
			vars:         hw{"stop_id": "70011", "sd": "2030-05-28", "ed": false},
			selector:     "stops.0.stop_times.#.arrival_time",
			selectExpect: []string{},
		},
	}
	return testcases
}
