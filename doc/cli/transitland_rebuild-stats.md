## transitland rebuild-stats

Rebuild statistics for feeds or specific feed versions

### Synopsis

Rebuild statistics for feeds or specific feed versions



```
transitland rebuild-stats [flags] [feeds...]
```

### Options

```
      --dburl string                       Database URL (default: $TL_DATABASE_URL)
      --fv-sha1 strings                    Feed version SHA1
      --fv-sha1-file string                Specify feed version IDs by SHA1 in file, one per line
      --fvid strings                       Rebuild stats for specific feed version ID
      --fvid-file string                   Specify feed version IDs in file, one per line; equivalent to multiple --fvid
  -h, --help                               help for rebuild-stats
      --stats strings                      Subset of stats to rebuild (default all); valid: file_infos,service_levels,service_windows,onestop_ids,geohash
      --storage string                     Storage destination; can be s3://... az://... or path to a directory
      --validation-report                  Save validation report
      --validation-report-storage string   Storage path for saving validation report JSON
      --workers int                        Worker threads (default 1)
```

### Options inherited from parent commands

```
      --cpuprofile file   Write a CPU profile to file
      --memprofile file   Write a heap profile to file at exit
```

### SEE ALSO

* [transitland](transitland.md)	 - transitland-lib utilities

