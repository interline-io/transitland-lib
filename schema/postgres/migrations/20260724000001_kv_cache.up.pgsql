BEGIN;

-- Shared key-value tier for kvcache. Unlogged because every cached byte is
-- regenerable (fetch cycles, gatekeeper HTTP), so crash-truncation self-heals
-- within a TTL/fetch cycle. Expiry is enforced on read; a periodic sweep
-- reclaims lapsed rows (expires_at NULL means no expiry).
CREATE UNLOGGED TABLE public.tl_kv_cache (
    key text PRIMARY KEY,
    value bytea,
    expires_at timestamptz
);
CREATE INDEX tl_kv_cache_expires_at_idx ON public.tl_kv_cache (expires_at) WHERE expires_at IS NOT NULL;

-- Field-addressable hash tier (HashStore), e.g. the GBFS bounding-box index.
CREATE UNLOGGED TABLE public.tl_kv_hash (
    hash_key text NOT NULL,
    field text NOT NULL,
    value text NOT NULL,
    PRIMARY KEY (hash_key, field)
);

COMMIT;
