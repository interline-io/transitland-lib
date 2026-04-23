## transitland validate

Validate a GTFS feed

### Synopsis

Validate a GTFS feed

The validate command performs a basic validation on a data source and writes the results to standard out.

```
transitland validate [flags] [<reader>]
```

### Examples

```
% transitland validate "https://www.bart.gov/dev/schedules/google_transit.zip"
% transitland validate -o - --include-entities "http://developer.trimet.org/schedule/gtfs.zip"
% transitland validate --dmfr feeds/wmata.com.dmfr.json --feed-id f-dqc-wmata~rail --secrets secrets.json
```

### Options

```
      --allow-ftp-fetch                    Allow fetching from FTP urls when --dmfr is used
      --allow-local-fetch                  Allow fetching from filesystem paths when --dmfr is used
      --allow-s3-fetch                     Allow fetching from S3 urls when --dmfr is used
      --best-practices                     Include Best Practices validations
      --dmfr string                        DMFR file providing feed URL and authorization config; used with --feed-id
      --error-limit int                    Max number of detailed errors per error group (default 1000)
      --error-threshold strings            Fail validation if file exceeds error percentage; format: 'filename:percent' or '*:percent' for default (e.g., 'stops.txt:5' or '*:10')
      --ext strings                        Include GTFS Extension
      --feed-id string                     Feed onestop ID for DMFR and secret lookup (requires --dmfr)
  -h, --help                               help for validate
      --include-entities                   Include GTFS entities in JSON output
      --include-route-geometries           Include route geometries in JSON output
      --include-service-levels             Include service levels in JSON output
  -o, --out string                         Write validation report as JSON to file; use '-' for stdout (implies -q)
  -q, --quiet                              Suppress log output
      --rt strings                         Include GTFS-RT proto message in validation report
      --rt-json                            Include GTFS-RT proto messages as JSON in validation report
      --save-fvid int                      Save report to feed version ID
      --secret-env stringArray             Specify secret from environment variable as feed_id:ENV_VAR or file.json:ENV_VAR (requires --dmfr and --feed-id)
      --secrets string                     Path to DMFR Secrets file (requires --dmfr and --feed-id)
      --url-type string                    URL type in DMFR feed.urls to validate (default "static_current")
      --validation-report                  Save static validation report in database
      --validation-report-storage string   Storage path for saving validation report JSON
```

### SEE ALSO

* [transitland](transitland.md)	 - transitland-lib utilities

