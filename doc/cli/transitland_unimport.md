## transitland unimport

Unimport feed versions

### Synopsis

Unimport feed versions

The `unimport` command deletes previously imported data from feed versions. The feed version record itself is not deleted. You may optionally specify removal of only schedule data, leaving routes, stops, etc. in place.

```
transitland unimport [flags] <fvids...>
```

### Options

```
      --dburl string          Database URL (default: $TL_DATABASE_URL)
      --dryrun                Dry run; print feeds that would be imported and exit
      --extra-table strings   Extra tables to delete feed_version_id
      --feed strings          Feed ID
      --fv-sha1 strings       Feed version SHA1
      --fv-sha1-file string   Specify feed version IDs by SHA1 in file, one per line
      --fvid-file string      Specify feed version IDs in file, one per line; equivalent to multiple --fvid
  -h, --help                  help for unimport
      --schedule-only         Unimport stop times, trips, transfers, shapes, and frequencies
```

### SEE ALSO

* [transitland](transitland.md)	 - transitland-lib utilities

###### Auto generated by spf13/cobra on 22-May-2025
