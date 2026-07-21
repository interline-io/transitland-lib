## transitland stats-remove-onestop-ids

Remove onestop_id stats for feed versions

### Synopsis

Remove onestop_id stats for feed versions

Deletes agency/route/stop onestop_id rows for the given feed versions; the feed versions are otherwise unaffected. The active and materialized feed versions are always skipped.

```
transitland stats-remove-onestop-ids [flags] <fvid>...
```

### Options

```
      --dburl string       Database URL (default: $TL_DATABASE_URL)
      --dry-run            Dry run; log the feed versions that would be affected and exit
      --fvid-file string   Read feed version IDs from a csv-like file (the feed_version_id column if the header names it, otherwise the first column of a header-less list of ids)
  -h, --help               help for stats-remove-onestop-ids
      --workers int        Worker threads (default 1)
```

### SEE ALSO

* [transitland](transitland.md)	 - transitland-lib utilities

