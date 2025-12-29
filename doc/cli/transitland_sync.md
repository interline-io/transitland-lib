## transitland sync

Sync DMFR files to database

### Synopsis

Sync DMFR files to database

Use '-' to read from stdin. New feeds are set to public by default; existing feeds retain their current public/private state unless --set-public or --set-private is specified.

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
      --set-private             Force all synced feeds to private (overrides default for new and existing feeds)
      --set-public              Force all synced feeds to public (overrides default for existing feeds)
```

### SEE ALSO

* [transitland](transitland.md)	 - transitland-lib utilities

