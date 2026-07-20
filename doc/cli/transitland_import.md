## transitland import

Import feed versions

### Synopsis

Import feed versions

Use after the `fetch` command

```
transitland import [flags] [feeds...]
```

### Options

```
      --activate                  Set as active feed version after import
      --allow-partial             Allow partial feeds missing normally-required files (agency, routes, trips, stop_times, calendar)
      --create-missing-shapes     Create missing Shapes from Trip stop-to-stop geometries
      --dburl string              Database URL (default: $TL_DATABASE_URL)
      --deduplicate-stop-times    Deduplicate StopTimes using Journey Patterns
      --dmfr string               Filter by feed IDs in DMFR file; equivalent to specifying feed IDs as arguments
      --dry-run                   Dry run; print feeds that would be imported and exit
      --error-threshold strings   Fail import if file exceeds error percentage; format: 'filename:percent' or '*:percent' for default (e.g., 'stops.txt:5' or '*:10')
      --ext strings               Include GTFS Extension
      --fail                      Exit with error code if any fetch is not successful
      --fv-sha1 strings           Feed version SHA1
      --fvid strings              Import specific feed version ID
      --fvid-file string          Read feed version IDs from a csv-like file (the feed_version_id column if present, else the first column; a non-numeric header row is ignored)
  -h, --help                      help for import
      --interpolate-stop-times    Interpolate missing StopTime arrival/departure values
      --latest                    Only import latest feed version available for each feed
      --limit int                 Import at most n feeds
      --normalize-timezones       Normalize timezones and apply default stop timezones based on agency and parent stops
      --simplify-calendars        Attempt to simplify CalendarDates into regular Calendars
      --simplify-shapes float     Simplify shapes with this tolerance (ex. 0.000005)
      --storage string            Storage location; can be s3://... az://... or path to a directory (default ".")
      --workers int               Worker threads (default 1)
```

### SEE ALSO

* [transitland](transitland.md)	 - transitland-lib utilities

