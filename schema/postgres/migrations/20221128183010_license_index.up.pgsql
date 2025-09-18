BEGIN;

CREATE INDEX ON current_feeds USING BTREE (((license->>'create_derived_product')::text));
CREATE INDEX ON current_feeds USING BTREE (((license->>'commercial_use_allowed')::text));
CREATE INDEX ON current_feeds USING BTREE (((license->>'share_alike_optional')::text));
CREATE INDEX ON current_feeds USING BTREE (((license->>'redistribution_allowed')::text));
CREATE INDEX ON current_feeds USING BTREE (((license->>'use_without_attribution')::text));

COMMIT;