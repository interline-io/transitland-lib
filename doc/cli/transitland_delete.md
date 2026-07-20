## transitland delete

Delete feed versions

### Synopsis

Delete feed versions



```
transitland delete [flags] <fvid>...
```

### Options

```
      --dburl string          Database URL (default: $TL_DATABASE_URL)
      --dryrun                Dry run; log the feed versions that would be deleted and exit
      --extra-table strings   Extra tables to delete feed_version_id
      --fvid-file string      Read feed version IDs from a csv-like file (the feed_version_id column if present, else the first column; a non-numeric header row is ignored)
  -h, --help                  help for delete
```

### SEE ALSO

* [transitland](transitland.md)	 - transitland-lib utilities

