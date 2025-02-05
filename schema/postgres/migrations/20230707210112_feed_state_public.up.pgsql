BEGIN;

ALTER TABLE feed_states ADD COLUMN public bool NOT NULL DEFAULT false;
CREATE INDEX ON feed_states(public);

COMMIT;