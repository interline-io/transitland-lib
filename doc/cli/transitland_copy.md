## transitland copy

Copy performs a basic copy from a reader to a writer.

### Synopsis

Copy performs a basic copy from a reader to a writer.

By default, any entity with errors will be skipped and not written to output.
This can be ignored with --allow-entity-errors to ignore simple errors and
--allow-reference-errors to ignore entity relationship errors, such as a
reference to a non-existent stop.

By default, the output order is determined by transitland-lib's streaming
architecture. It generally preserves the input order, although some records
may be reordered to maintain associations (such as ensuring parent stops are
processed before child stops).

Output can be automatically sorted using --standardized-sort (asc or desc).
This is an optional feature and is off by default. When enabled, it applies
an opinionated, standardized sort order to CSV files, which is useful for
consistent diffing and human readability. By default, it uses logical primary
GTFS columns (e.g., stop_id for stops.txt), but specific columns can be
provided with --standardized-sort-columns.

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
      --standardized-sort string            Standardized sort order for CSV files (asc, desc, or none)
      --standardized-sort-columns strings   Comma-separated list of columns to sort by (optional; if empty, defaults are used)
      --write-extra-columns                 Include extra columns in output
      --write-extra-files                   Copy additional files found in source to destination
```

### SEE ALSO

* [transitland](transitland.md)	 - transitland-lib utilities

