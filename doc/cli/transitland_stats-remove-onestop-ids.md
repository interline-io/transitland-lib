## transitland stats-remove-onestop-ids

Remove onestop_id stats for feed versions

### Synopsis

Remove onestop_id stats for feed versions

Deletes agency/route/stop onestop_id rows for the given feed versions; the feed versions are otherwise unaffected. The active and materialized feed versions are always skipped.

```
transitland stats-remove-onestop-ids [flags]
```

### Options

```
      --dburl string       Database URL (default: $TL_DATABASE_URL)
      --dryrun             Dry run; log the feed versions that would be affected and exit
      --fvid strings       Remove onestop_id stats for specific feed version ID
      --fvid-file string   Specify feed version IDs in file, one per line; equivalent to multiple --fvid
  -h, --help               help for stats-remove-onestop-ids
      --workers int        Worker threads (default 1)
```

### SEE ALSO

* [transitland](transitland.md)	 - transitland-lib utilities

