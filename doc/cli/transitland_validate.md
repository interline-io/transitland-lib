## transitland validate

Validate a GTFS feed

### Synopsis

Validate a GTFS feed

The validate command performs a basic validation on a data source and writes the results to standard out.

```
transitland validate [flags] <reader>
```

### Examples

```
% transitland validate "https://www.bart.gov/dev/schedules/google_transit.zip"
% transitland validate -o - --include-entities "http://developer.trimet.org/schedule/gtfs.zip"
```

### Options

```
      --best-practices                     Include Best Practices validations
      --error-limit int                    Max number of detailed errors per error group (default 1000)
      --error-threshold strings            Fail validation if file exceeds error percentage; format: 'filename:percent' or '*:percent' for default (e.g., 'stops.txt:5' or '*:10')
      --ext strings                        Include GTFS Extension
  -h, --help                               help for validate
      --include-entities                   Include GTFS entities in JSON output
      --include-route-geometries           Include route geometries in JSON output
      --include-service-levels             Include service levels in JSON output
  -o, --out string                         Write validation report as JSON to file; use '-' for stdout (implies -q)
  -q, --quiet                              Suppress log output
      --rt strings                         Include GTFS-RT proto message in validation report
      --rt-json                            Include GTFS-RT proto messages as JSON in validation report
      --save-fvid int                      Save report to feed version ID
      --validation-report                  Save static validation report in database
      --validation-report-storage string   Storage path for saving validation report JSON
```

### SEE ALSO

* [transitland](transitland.md)	 - transitland-lib utilities

