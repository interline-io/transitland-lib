## transitland dbmigrate

Perform database migrations

### Synopsis

Perform database migrations

Subcommands: up (apply pending migrations); check (exit non-zero if the database is dirty or behind this binary's embedded migrations, for use as a deploy gate).

```
transitland dbmigrate [flags] <up|check>
```

### Options

```
      --dburl string   Database URL (default: $TL_DATABASE_URL)
  -h, --help           help for dbmigrate
```

### SEE ALSO

* [transitland](transitland.md)	 - transitland-lib utilities

