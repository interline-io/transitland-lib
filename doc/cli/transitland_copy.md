## transitland copy

Copy performs a basic copy from a reader to a writer.

### Synopsis

Copy performs a basic copy from a reader to a writer.

Entities with errors are skipped by default; use --allow-entity-errors and --allow-reference-errors to override.

Output preserves input order (with exceptions for stop relationships). Pass --standardized-sort (asc|desc) to apply an opinionated GTFS sort by primary keys, or override columns with --standardized-sort-columns.

```
transitland copy [flags] <reader> <writer>
```

### Examples

```

% transitland copy --allow-entity-errors "https://www.bart.gov/dev/schedules/google_transit.zip" output.zip

% unzip -p output.zip agency.txt
agency_id,agency_name,agency_url,agency_timezone,agency_lang,agency_phone,agency_fare_url,agency_email
BART,Bay Area Rapid Transit,https://www.bart.gov/,America/Los_Angeles,,510-464-6000,,

```

### Options

```
      --allow-entity-errors                 Allow entities with errors to be copied
      --allow-reference-errors              Allow entities with reference errors to be copied
      --create                              Create a basic database schema if none exists
      --error-limit int                     Max number of detailed errors per error group (default 1000)
      --ext strings                         Include GTFS Extension
      --fvid int                            Specify FeedVersionID when writing to a database
  -h, --help                                help for copy
      --standardized-sort string            Standardized sort order for CSV files (asc or desc; empty = no sort)
      --standardized-sort-columns strings   Comma-separated list of columns to sort by (optional; if empty, defaults are used)
      --write-extra-columns                 Include extra columns in output
      --write-extra-files                   Copy additional files found in source to destination
```

### SEE ALSO

* [transitland](transitland.md)	 - transitland-lib utilities

