## transitland unimport

Unimport feed versions

### Synopsis

Unimport feed versions

The `unimport` command deletes previously imported data from feed versions. The feed version record itself is not deleted. You may optionally specify removal of only schedule data, leaving routes, stops, etc. in place.

```
transitland unimport [flags] <fvid>...
```

### Options

```
      --dburl string          Database URL (default: $TL_DATABASE_URL)
      --dry-run               Dry run; log the feed versions that would be unimported and exit
      --extra-table strings   Extra tables to delete feed_version_id
      --fvid-file string      Read feed version IDs from a csv-like file (the feed_version_id column if the header names it, otherwise the first column of a header-less list of ids)
  -h, --help                  help for unimport
      --schedule-only         Unimport stop times, trips, transfers, shapes, and frequencies
      --workers int           Worker threads (default 1)
```

### SEE ALSO

* [transitland](transitland.md)	 - transitland-lib utilities

