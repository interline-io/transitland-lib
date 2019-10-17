ALTER TABLE current_feeds ALTER COLUMN urls TYPE JSONB USING hstore_to_json(urls);
ALTER TABLE old_feeds ALTER COLUMN urls TYPE JSONB USING hstore_to_json(urls);

ALTER TABLE current_feeds ALTER COLUMN "authorization" TYPE JSONB USING hstore_to_json("authorization");
ALTER TABLE old_feeds ALTER COLUMN "authorization" TYPE JSONB USING hstore_to_json("authorization");

