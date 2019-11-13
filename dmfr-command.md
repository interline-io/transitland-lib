# DMFR Command Help

Gotransit DMFR provides the foundation for [Transitland v2](https://transit.land/news/2019/10/17/tlv2.html). Gotransit provides both a cli interface for managing a DMFR instance, as well as a library.

Coming soon!

DMFR Subcommands:
- [sync](#sync-command)
- [fetch](#fetch-command)
- [import](#import-command)

## sync command

```bash
  $ ./gotransit dmfr sync -h
Usage: sync <filenames...>
  -dburl string
    	Database URL (default: $DMFR_DATABASE_URL)
```

## fetch command

```bash
$ ./gotransit dmfr fetch -h
Usage: fetch [feedids...]
  -allow-duplicate-contents
    	Allow duplicate internal SHA1 contents
  -dburl string
    	Database URL (default: $DMFR_DATABASE_URL) 
  -gtfsdir string
    	GTFS Directory (default ".")
  -limit int
    	Maximum number of feeds to fetch
  -workers int
    	Worker threads (default 1)
```

## import command

```bash
$ ./gotransit dmfr import -h
Usage: import [feedids...]
  -date string
    	Service on date
  -dburl string
    	Database URL (default: $DMFR_DATABASE_URL) 
  -dryrun
    	Dry run; print feeds that would be imported and exit
  -ext value
    	Include GTFS Extension
  -gtfsdir string
    	GTFS Directory (default ".")
  -limit uint
    	Import at most n feeds
  -workers int
    	Worker threads (default 1)
```

