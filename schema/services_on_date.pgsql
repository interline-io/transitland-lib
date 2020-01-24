CREATE OR REPLACE FUNCTION services_on_date(fvid bigint, service_date date)
RETURNS SETOF gtfs_calendars AS $$
WITH calendar_ids AS (
    SELECT
        id
    FROM
        gtfs_calendars
    WHERE
        start_date <= service_date
        AND end_date >= service_date
        AND feed_version_id = fvid
        AND id NOT IN (
            SELECT 
                service_id 
            FROM 
                gtfs_calendar_dates 
            WHERE 
                date = service_date AND 
                exception_type = 2 AND 
                feed_version_id = fvid
        )
        AND (CASE EXTRACT(isodow FROM service_date)
            WHEN 1 THEN monday = 1
            WHEN 2 THEN tuesday = 1
            WHEN 3 THEN wednesday = 1
            WHEN 4 THEN thursday = 1
            WHEN 5 THEN friday = 1
            WHEN 6 THEN saturday = 1
            WHEN 7 THEN sunday = 1
        END)
    UNION
    SELECT
        service_id
    FROM
        gtfs_calendar_dates
    WHERE
        date = service_date
        AND exception_type = 1
        AND feed_version_id = fvid
)
SELECT
    DISTINCT ON (id)
    gtfs_calendars.*
FROM
    calendar_ids
INNER JOIN gtfs_calendars ON gtfs_calendars.id = calendar_ids.id
$$ LANGUAGE sql STABLE;

