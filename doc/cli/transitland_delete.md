## transitland delete

Delete feed versions

### Synopsis

Delete feed versions

The `delete` command soft deletes a feed version and removes its remaining rows. The feed version must already be unimported: run `unimport` first.

```
transitland delete [flags] <fvid>
```

### Options

```
      --dburl string          Database URL (default: $TL_DATABASE_URL)
      --dryrun                Dry run; print feeds that would be imported and exit
      --extra-table strings   Extra tables to delete feed_version_id
  -h, --help                  help for delete
```

### SEE ALSO

* [transitland](transitland.md)	 - transitland-lib utilities

