# GTFS Flex GraphQL Schema Plan

This document outlines the changes to the GraphQL schema to support the officially adopted GTFS Flex specification.

## 1. New Types for GTFS Flex Entities

We are adding four new types to represent the core Flex files. These map directly to the official spec.

### `Location` (`locations.geojson`)
*   Represents a demand-responsive zone (polygon).
*   **Fields**: `location_id`, `stop_name`, `geometry` (Polygon/MultiPolygon), etc.
*   **Relationships**:
    *   `stop_times`: Returns `[FlexStopTime!]!` (navigating from a Location to the service that serves it).
    *   `feed_version`: Link back to the parent feed version.

### `BookingRule` (`booking_rules.txt`)
*   Represents booking information for demand-responsive services.
*   **Fields**: `booking_type`, `prior_notice_duration_min/max`, `message`, `phone_number`, `booking_url`, etc.
*   **Relationships**:
    *   `prior_notice_service`: Links to a `Calendar` object (if `prior_notice_service_id` is present).

### `LocationGroup` (`location_groups.txt`)
*   Represents a group of stops (e.g., a "downtown zone" containing multiple specific stops).
*   **Fields**: `location_group_id`, `location_group_name`.
*   **Relationships**:
    *   `stops`: Returns the list of `Stop` entities that belong to this group.

### `LocationGroupStop` (`location_group_stops.txt`)
*   Represents the many-to-many relationship between `LocationGroup` and `Stop`.

## 2. The "Parallel Types" Strategy for Stop Times

To support GTFS Flex without breaking existing clients that expect `Stop!` (non-nullable) on stop times, we are introducing a parallel type system:

### `StopTime` (Existing)
*   **Contract**: `stop: Stop!` (Non-nullable). This type **only** returns stop times that reference a standard fixed stop.
*   **Updates**: Added Flex fields (`pickup_booking_rule`, `start_pickup_drop_off_window`, etc.) because fixed stops *can* have booking rules.
*   **Removed**: Removed `location` and `location_group` fields from this type to keep it strictly for fixed-stop scenarios.

### `FlexStopTime` (New)
*   **Contract**: `location: Location` and `location_group: LocationGroup` (Nullable, but at least one will be present). This type **only** returns stop times that reference a Flex location or group.
*   **Fields**: Mirrors `StopTime` exactly, but replaces the `stop` field with `location` and `location_group`.

## 3. Updates to `Trip`

The `Trip` type now has two separate fields to access its schedule:

*   **`stop_times`**: Returns `[StopTime]!`.
    *   *Behavior*: Returns only the fixed-route portion of the trip (standard stops). Legacy clients see exactly what they expect.
*   **`flexible_stop_times`**: Returns `[FlexStopTime!]!`.
    *   *Behavior*: Returns only the flexible portion of the trip (zones and groups). New clients use this to render Flex zones.

## 4. Updates to `FeedVersion`

*   **Added `locations` field**:
    *   `locations(limit: Int, where: LocationFilter): [Location!]!`
    *   This is the entry point for applications to fetch all GeoJSON zones for a specific feed version.

## 5. Updates to `Stop`

*   **Added `location_groups` field**:
    *   Allows querying a standard stop and seeing which Location Groups it belongs to.

## 6. New Input Filters

*   **`LocationFilter`**:
    *   Added to allow filtering locations by `ids` (internal DB ID) or `location_id` (GTFS ID).
