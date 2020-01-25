CREATE OR REPLACE FUNCTION activate_feed_version(fvid bigint) RETURNS integer AS $$
DECLARE
    fid integer;
    fvid ALIAS for $1;
BEGIN

SELECT feed_id INTO STRICT fid FROM feed_versions WHERE feed_versions.id = fvid;
RAISE NOTICE 'activate_feed_version fid: % fvid: %', fid, fvid;

RAISE NOTICE '... setting feed_states feed_version_id';
UPDATE feed_states SET feed_version_id = fvid WHERE feed_states.feed_id = fid;

RETURN 0;
END;
$$ LANGUAGE plpgsql;
