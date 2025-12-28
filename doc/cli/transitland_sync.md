## transitland sync

Sync DMFR files to database

### Synopsis

Sync DMFR files to database

Use '-' to read from stdin.

```
transitland sync [flags] <filenames...>
```

### Examples

```

  # Sync from a file
  transitland sync feeds.dmfr

  # Sync from a directory of GTFS files
  transitland dmfr from-dir ./gtfs-files/ | transitland sync -

```

### Options

```
      --dburl string            Database URL (default: $TL_DATABASE_URL)
  -h, --help                    help for sync
      --hide-unseen             Hide unseen feeds
      --hide-unseen-operators   Hide unseen operators
```

### SEE ALSO

* [transitland](transitland.md)	 - transitland-lib utilities

