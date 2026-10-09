# Matching realtime data to trips

Transitland attaches GTFS Realtime data (trip updates, vehicle positions and service alerts) to the trips it describes. A trip can run on many days, so realtime data is matched to a **run**: one trip on one service date. This page describes how a message is matched to a run, and which run's data each API field returns.

## Which run a message describes

Every realtime message describes exactly one run.

- **A message that names a `start_date`** in its trip descriptor describes the run on that date.
- **A message that names none** describes the trip's **current run**: the run going now, or failing that, the nearest of yesterday's, today's and tomorrow's.

A run lasts from its trip's first scheduled departure to its last scheduled arrival. A frequency-based trip runs from its first start to its last start plus one trip's length, counting a start at `end_time` as departures do. Times count in the agency's timezone from noon minus twelve hours on the service date, as GTFS defines them.

A run that has just ended stays current until another run is nearer, so a train running late keeps its updates after its scheduled end. A trip whose span isn't known, such as a trip added in real time, is on today's run.

For example, for an undated message:

| Now | The trip runs | The message describes |
|---|---|---|
| Tuesday 08:30 | 08:00 to 09:00 | Tuesday's run, which is going |
| Tuesday 09:20 | 08:00 to 09:00 | Tuesday's run, which ended 20 minutes ago and is still the nearest |
| Tuesday 23:50 | 00:10 to 00:50 | Wednesday's run, which starts in 20 minutes |
| Wednesday 00:10 | 23:40 to 24:31 | Tuesday's run, which is still going past midnight |
| Wednesday 18:00 | 23:40 to 24:31 | Wednesday's run, which starts in under six hours |

If a feed sends several updates for the same run, the last one wins.

## Which run a trip stands for

A trip found on a date is the run on that date. That includes:

- the trip of a stop time with a `service_date`, as in departures
- trips from `trips`, `Route.trips` or `FeedVersion.trips` queried by `service_date`, `service_dates` or `dates`
- a route pattern's `trips` and `timetable` when the pattern has a `service_date`
- a vehicle position's trip, on the `start_date` the vehicle names
- a trip added in real time, on the `start_date` its trip update names

A trip found any other way, for example by `trip_id` alone, is its current run.

A route pattern's `representative_trip`, and its `trips` when the pattern has no `service_date`, stand for the pattern rather than any run. They carry no realtime data.

## What each field returns

These fields return only the realtime data on the trip's run:

- `StopTime.arrival`, `departure` and `schedule_relationship`: the trip update for the stop time's run.
- `Trip.stop_times`: on a trip that is one run, stop times on that run's service date, with that run's trip update.
- `Trip.alerts`: alerts whose selector names this trip and its run.
- `Trip.vehicle_position`: a vehicle running this trip's run.
- `Trip.schedule_relationship` and `Trip.timestamp`: from the trip update for the trip's run.

A trip found without a date has no service date for its stop times, so a trip update that gives only delays, not times, produces no estimates on its `stop_times`.

## Dates under use_service_window

With `use_service_window`, a date outside the feed version's service window is answered from the same weekday of its fallback week. The answer still reports the requested date:

- Departures report the requested `service_date` and `date`, and their scheduled `_unix`, `_utc` and `_local` times fall on that date.
- Each service day of a departures query, including the previous day for trips past midnight, is moved into the fallback week on its own.
- `RouteStopPattern.service_date` reports the requested date.

Realtime data is then matched to the run on the requested date. On a feed whose schedule has lapsed, today's departures still carry today's realtime data.

The top-level `trips` query doesn't apply `use_service_window`, because it spans feed versions with different windows. Use `Route.trips` or `FeedVersion.trips` instead.

## Relation to the GTFS Realtime spec

The spec defines `start_date` as the service date of a trip instance, and that's how it is matched. The spec doesn't say which instance an undated message means; "the current run" is our reading of it, and one rule covers trip updates, vehicle positions and alerts alike.

The spec requires an alert's trip selector to resolve to a single trip instance. An undated alert about a future run therefore lands on the current run, and a producer should name the date.

## Not yet handled

- Delays aren't yet carried forward to later stops the way the spec describes, and the per-stop `SKIPPED` and `NO_DATA` relationships are ignored.
- Scheduled times in API responses use the stop's timezone and midnight instead of the agency's timezone and noon minus twelve hours.
- `NEW` and `DUPLICATED` trips, and trips identified without a `trip_id`, aren't matched. Frequency-based trips aren't matched by `start_time`.
- Trips added in real time are listed at a stop whatever date and time window the query asks for.
