CREATE OR REPLACE FUNCTION calculate_route_headways(fvid bigint) RETURNS integer AS $$
DECLARE
    fvid ALIAS for $1;
BEGIN

RAISE NOTICE 'calculate_route_headways: %', fvid;

RAISE NOTICE '... deleting route_headways entries';
DELETE FROM route_headways where feed_version_id = fvid;

RAISE NOTICE '... inserting route_headways entries';
INSERT INTO route_headways(
    feed_version_id,
    route_id,
    selected_stop_id,
    service_id,
    direction_id,
    headway_secs
)
select
    gtfs_routes.feed_version_id,
    gtfs_routes.id as route_id,
    gtfs_stops.id as selected_stop_id,
    gtfs_calendars.id as service_id,
    a.direction_id as direction_id,
    (
        case
        when b.headway_secs < 30 then null
        when b.count < 4 then null
        else b.headway_secs
        end
    ) as headway_secs
from
    gtfs_routes
join lateral (
    select
        distinct on(gtfs_trips.route_id)
        gtfs_trips.route_id,
        gtfs_trips.service_id,
        gtfs_trips.direction_id,
        gtfs_stop_times.stop_id,
        count(*)
    from
        gtfs_trips
    inner join gtfs_stop_times on gtfs_stop_times.trip_id = gtfs_trips.id
    where gtfs_trips.route_id = gtfs_routes.id
    group by (gtfs_trips.route_id,gtfs_trips.service_id,gtfs_trips.direction_id,gtfs_stop_times.stop_id)
    order by gtfs_trips.route_id,count desc
    limit 1
) a on true
left join lateral (
    select
        percentile_disc(0.75) within group (order by hw) as headway_secs,
        count(*)
    from
    (
        select 
            distinct on (gtfs_trips.id)
            departure_time - lag(departure_time,1) OVER (order by departure_time) as hw
        from gtfs_stop_times 
        inner join gtfs_trips on gtfs_trips.id = gtfs_stop_times.trip_id 
        where gtfs_trips.route_id = gtfs_routes.id and 
            gtfs_trips.direction_id = a.direction_id and 
            gtfs_trips.service_id = a.service_id and 
            gtfs_stop_times.stop_id = a.stop_id and
            departure_time >= 21600 and departure_time <= 36000
        order by gtfs_trips.id, gtfs_stop_times.stop_sequence
    ) c
) b on true
inner join gtfs_stops on gtfs_stops.id = a.stop_id
inner join gtfs_calendars on gtfs_calendars.id = a.service_id
where gtfs_routes.feed_version_id = fvid;

RETURN 0;
END;
$$ LANGUAGE plpgsql;
