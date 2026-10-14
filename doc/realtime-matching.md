# Matching realtime data to trips

Transitland attaches GTFS Realtime data (trip updates, vehicle positions and service alerts) to the trips it describes. A trip can run on many days, so realtime data is matched to a **run**: one trip on one service date. This page describes how a message is matched to a run, and which run's data each API field returns.

## Payloads too old to match

Trip updates and vehicle positions are matched only from a payload whose header timestamp is within an hour of now. A live feed's payload is a few minutes old at most. Each fetch stores the payload again, so a producer that keeps serving old content keeps it held for as long as the fetches succeed, and without this rule its delays, cancellations and vehicles would keep applying long after they stopped being true. The store's 24-hour expiry applies only once fetches stop. A header more than an hour ahead of the clock, as from a producer stamping milliseconds or with a skewed clock, is stale too. A payload without a header timestamp is matched whatever its age.

Service alerts aren't held to this, since an alert carries its own active periods, and many feeds serve alerts from files that never update their header. The latest realtime download serves a feed's last payload however old it is, and the Finder's `GetMessage` reports whether a payload is stale, so other readers can decide for themselves.

## Which run a message describes

Every realtime message describes exactly one run.

- **A message that names a `start_date`** in its trip descriptor describes the run on that date.
- **A message that names none** describes the trip's **current run**: the run going now, or failing that, the nearest of yesterday's, today's and tomorrow's.

A run lasts from its trip's first scheduled departure to its last scheduled arrival. A frequency-based trip runs from its first start to its last start plus one trip's length, with a start every `headway_secs` from `start_time`, up to but not including `end_time`. Times count in the agency's timezone from noon minus twelve hours on the service date, as GTFS defines them.

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

A `DUPLICATED` message names the trip it copies but describes a new run, so it never describes one of that trip's runs, whether it's a trip update, a vehicle position or an alert. Neither does an `ADDED` message that names a scheduled trip, the form duplicates took before `DUPLICATED` existed. An `ADDED` message describes only a trip added in real time, which every other message naming its `trip_id` describes too, since alerts and vehicle positions rarely repeat `ADDED`. A vehicle running a copy of a trip has no trip until copies are shown as runs of their own, and a vehicle running a trip added in real time has that trip.

## Which stop time an update describes

A trip update's stop time updates are placed on the run's stop times all at once, in order along the trip: each is looked for at or after the stop time the previous one was placed on. A stop time update that names a `stop_sequence` describes the stop time with that `stop_sequence`; one that names only a `stop_id` describes the next stop time at that stop, which picks the right visit on a loop that serves a stop twice. One that names both takes the stop time that agrees on both. Failing that, it takes the next stop time at its `stop_id`, for a feed whose `stop_sequence` values aren't the static feed's, as after a renumbering; failing that, the stop time at its `stop_sequence`, for an update whose `stop_id` is an assigned platform the schedule doesn't have. A stop time update that names no stop time from there on, or that comes out of order, is left out.

Departures, a trip's stop times, and the router's trips all match this way, since matching asks the static data for the whole trip rather than taking the stop times a query happens to hold.

## Delays and estimates

Each stop time's estimated arrival and departure come from its own stop time update when it has one: the departure event, or else the arrival event, for the departure, and the other way round for the arrival. An event's time is the estimate; an event with only a delay adds it to the scheduled time.

A stop time the update doesn't predict takes the delay carried from earlier in the trip, as the GTFS Realtime spec describes: the trip update's own `delay` until the first stop time update, then the delay given at the nearest earlier stop time update, or the delay its time implies against the schedule. A `SKIPPED` stop has no estimate, though a delay its update gives still carries; a `NO_DATA` stop ends the carry, so the stops after it have no estimate until an update gives one.

Scheduled times count from noon minus twelve hours on the service date in the agency's timezone, as GTFS defines them, so an estimate on the day clocks change is still right. A trip found without a date has no scheduled instants, so only the times its update gives become estimates: delays, given or carried, estimate nothing.

The GraphQL API's `StopTime.arrival` and `departure` report these estimates, and the router boards and alights by the estimated departures. `StopTime.schedule_relationship` reports `SKIPPED` and `NO_DATA` for such stops, `CANCELED` or `DELETED` for every stop of such a trip, and otherwise the trip's relationship, or `SCHEDULED` for a stop time with any other realtime data, such as a carried delay.

## Which run a trip stands for

A trip found on a date is the run on that date. That includes:

- the trip of a stop time with a `service_date`, as in departures
- trips from `trips`, `Route.trips`, `FeedVersion.trips` or `Shape.trips` queried by `service_date`, `service_dates` or `dates`, and from all but the top-level `trips` queried by `relative_date`
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

`StopTime.arrival` and `departure` give `scheduled` and `estimated` as times of the service day in the agency's clock, so `estimated` less `scheduled` is `estimated_delay`, and give `scheduled_local` and `estimated_local` in the stop's timezone. Without a schedule, `estimated` is a clock time in the stop's timezone.

## Dates under use_service_window

With `use_service_window`, a date outside the feed version's service window is answered from the same weekday of its fallback week. The answer still reports the requested date:

- Departures report the requested `service_date` and `date`, and their scheduled `_unix`, `_utc` and `_local` times fall on that date.
- Each service day of a departures query, including the previous day for trips past midnight, is moved into the fallback week on its own.
- `RouteStopPattern.service_date` reports the requested date.

Realtime data is then matched to the run on the requested date. On a feed whose schedule has lapsed, today's departures still carry today's realtime data.

The top-level `trips` query doesn't apply `use_service_window` or `relative_date`, because it spans feed versions with different windows and timezones. Use `Route.trips`, `FeedVersion.trips` or `Shape.trips` instead.

## Relation to the GTFS Realtime spec

The spec defines `start_date` as the service date of a trip instance, and that's how it is matched. The spec doesn't say which instance an undated message means; "the current run" is our reading of it, and one rule covers trip updates, vehicle positions and alerts alike.

The spec requires an alert's trip selector to resolve to a single trip instance. An undated alert about a future run therefore lands on the current run, and a producer should name the date.

## Not yet handled

- A run described by several realtime feeds takes the first feed's trip update for everything.
- `NEW` and `DUPLICATED` trips aren't shown as runs of their own, and trips identified without a `trip_id` aren't matched. Frequency-based trips aren't matched by `start_time`.
- Trips added in real time are listed at a stop whatever date and time window the query asks for, and only on a route the feed version has.
