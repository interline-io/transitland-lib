# SCMTD: `modified_trip` alongside legacy TripDescriptor fields (W100)

**Producer:** Santa Cruz Metropolitan Transit District
**Feed:** `https://rt.scmetro.org/gtfsrt/trips`
**Observed:** 2026-05-18
**Validator code:** `W100` (transitland-lib `rt/errors.go`)

## What we see

Every `TripUpdate` whose `trip` carries a `modified_trip` selector also sets `trip.route_id`. Example:

```json
{
  "trip_update": {
    "trip": {
      "route_id": "2",
      "schedule_relationship": "SCHEDULED",
      "modified_trip": {
        "modifications_id": "trip_modifications_500286",
        "affected_trip_id": "1206020"
      }
    },
    ...
  }
}
```

At the time of capture, 9 of 139 TripUpdates carried `modified_trip`; all 9 also set `route_id`. None set `trip_id`, `direction_id`, `start_time`, or `start_date` — only `route_id` is duplicated.

## Why it's wrong

The [GTFS-RT TripDescriptor spec](https://gtfs.org/documentation/realtime/feed-entities/trip-modifications/) says:

> When `modified_trip` is provided, the fields `trip_id`, `route_id`, `direction_id`, `start_time`, and `start_date` MUST be empty. The affected trip is identified entirely through `modified_trip.affected_trip_id` and the linked `TripModifications` entity.

SCMTD's feed identifies the affected trip correctly via `modified_trip.affected_trip_id` (e.g. `"1206020"`), but the redundant `route_id` makes the descriptor ambiguous: a consumer that prefers legacy fields would route the update under `route_id=2` without ever resolving the modification, and a consumer that prefers `modified_trip` would silently disagree.

## Impact on transitland-lib

- `rt/validator.go` emits **`W100`** for these entries: *"modified_trip is set alongside legacy TripDescriptor identifier fields."* This is a warning rather than an error — the trip is still resolvable.
- Indexing in `server/finders/rtfinder/source.go` falls back to `affected_trip_id` when `trip_id` is empty, so the update is correctly associated with static trip `1206020`. The duplicated `route_id` is ignored.

## Recommendation to producer

Drop `route_id` from any TripUpdate whose `trip` uses `modified_trip`. The `TripModifications` entity (`id: trip_modifications_500286`) already provides everything a downstream consumer needs to look up the route via the static `affected_trip_id`.
