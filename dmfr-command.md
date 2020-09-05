# DMFR Command Help

Transitland DMFR provides the foundation for [Transitland v2](https://transit.land/news/2019/10/17/tlv2.html). Transitland provides both a cli interface for managing a DMFR instance, as well as a library. The cli interface can synchronize DMFR files to feed records in the database, fetch each feed and create feed version records, and import the GTFS data from each feed version into the database.

DMFR Subcommands:
- [sync](#sync-command)
- [fetch](#fetch-command)
- [import](#import-command)

## sync command

```bash
% transitland dmfr sync -h
Usage: sync <Filenames...>
  -dburl string
    	Database URL (default: $DMFR_DATABASE_URL)
  -hide-unseen
    	Hide unseen feeds
```

## fetch command

```bash
% transitland dmfr fetch -h
Usage: fetch [feed_id...]
  -dburl string
    	Database URL (default: $DMFR_DATABASE_URL)
  -dry-run
    	Dry run; print feeds that would be imported and exit
  -feed-url string
    	Manually fetch a single URL; you must specify exactly one feed_id
  -fetched-at string
    	Manually specify fetched_at value, e.g. 2020-02-06T12:34:56Z
  -gtfsdir string
    	GTFS Directory (default ".")
  -ignore-duplicate-contents
    	Allow duplicate internal SHA1 contents
  -limit int
    	Maximum number of feeds to fetch
  -s3 string
    	Upload GTFS files to S3 bucket/prefix
  -secrets string
    	Path to DMFR Secrets file
  -workers int
    	Worker threads (default 1)
```

## import command

```bash
% transitland dmfr import -h
Usage: import [feedids...]
  -activate
    	Set as active feed version after import
  -create-missing-shapes
    	Create missing Shapes from Trip stop-to-stop geometries
  -date string
    	Service on date
  -dburl string
    	Database URL (default: $DMFR_DATABASE_URL)
  -dryrun
    	Dry run; print feeds that would be imported and exit
  -ext value
    	Include GTFS Extension
  -fetched-since string
    	Fetched since
  -fvid value
    	Import specific feed version ID
  -fvid-file string
    	Specify feed version IDs in file, one per line; equivalent to multiple --fvid
  -gtfsdir string
    	GTFS Directory (default ".")
  -interpolate-stop-times
    	Interpolate missing StopTime arrival/departure values
  -latest
    	Only import latest feed version available for each feed
  -limit int
    	Import at most n feeds
  -s3 string
    	Get GTFS files from S3 bucket/prefix
  -workers int
    	Worker threads (default 1)
```

