# GTFS-Flex: Duplicate Geography ID (Stop vs Location)

This test validates that geography IDs must be unique across:
- `stops.stop_id`
- `locations.geojson` feature `id`
- `location_groups.location_group_id`

## Test Case

The base feed contains a stop with `stop_id = "stop1"`.

This overlay adds a `locations.geojson` with a feature that has `id = "stop1"`.

## Expected Error

`FlexGeographyIDDuplicateError` - The geography ID "stop1" is duplicated across stops.txt and locations.geojson.

## Specification Reference

Per GTFS-Flex specification, these three ID types share the same namespace for referencing in `stop_times.txt` (via `stop_id` or `location_id` fields).

