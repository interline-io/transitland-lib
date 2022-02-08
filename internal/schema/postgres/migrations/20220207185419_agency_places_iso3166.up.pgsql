BEGIN;

ALTER TABLE tl_agency_places ADD COLUMN adm1iso character varying;
ALTER TABLE tl_agency_places ADD COLUMN adm0iso character varying;

COMMIT;