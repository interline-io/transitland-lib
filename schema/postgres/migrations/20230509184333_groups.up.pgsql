BEGIN;

CREATE TABLE tl_groups (
    id bigserial primary key,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    group_name text
);

CREATE TABLE tl_tenants (
    id bigserial primary key,
    created_at timestamp without time zone DEFAULT now() NOT NULL,
    updated_at timestamp without time zone DEFAULT now() NOT NULL,
    tenant_name text
);

COMMIT;