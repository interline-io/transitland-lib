UPDATE feed_states
SET feed_version_id = a.feed_version_id
FROM
(
    SELECT 
        DISTINCT ON(feed_id)
        feed_versions.feed_id,
        feed_versions.id as feed_version_id
    FROM feed_version_gtfs_imports 
    INNER JOIN feed_versions ON feed_versions.id = feed_version_gtfs_imports.feed_version_id
    WHERE success = true AND feed_versions.earliest_calendar_date <= NOW()
    ORDER BY feed_versions.feed_id, feed_versions.fetched_at DESC
) a
WHERE feed_states.feed_id = a.feed_id;
