## transitland stats-rebuild

Rebuild statistics for feed versions

### Synopsis

Rebuild statistics for feed versions

With no feed version ids given, rebuilds stats for all feed versions.

```
transitland stats-rebuild [flags] [fvid...]
```

### Options

```
      --dburl string                       Database URL (default: $TL_DATABASE_URL)
      --dry-run                            Dry run; log the feed versions that would be rebuilt and exit
      --fvid-file string                   Read feed version IDs from a csv-like file (the feed_version_id column if the header names it, otherwise the first column of a header-less list of ids)
  -h, --help                               help for stats-rebuild
      --stats strings                      Subset of stats to rebuild (default all); valid: file_infos,service_levels,service_windows,onestop_ids,geohash
      --storage string                     Storage destination; can be s3://... az://... or path to a directory
      --validation-report                  Save validation report
      --validation-report-storage string   Storage path for saving validation report JSON
      --workers int                        Worker threads (default 1)
```

### SEE ALSO

* [transitland](transitland.md)	 - transitland-lib utilities

