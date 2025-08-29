# Interline Transitland <!-- omit in toc -->

`transitland-lib` is a library and command-line tool for reading, writing, and processing transit data in [GTFS](http://gtfs.org) and related formats. The library is structured as a set of data sources, filters, and transformations that can be mixed together in a variety of ways to create processing pipelines. The library supports the [DMFR](https://github.com/transitland/distributed-mobility-feed-registry) format to describe feed resources.

![Test & Release](https://github.com/interline-io/transitland-lib/workflows/Test%20&%20Release/badge.svg) [![GoDoc](https://godoc.org/github.com/interline-io/transitland-lib/tl?status.svg)](https://godoc.org/github.com/interline-io/transitland-lib/tl) ![Go Report Card](https://goreportcard.com/badge/github.com/interline-io/transitland-lib)

## Table of Contents <!-- omit in toc -->
<!-- to update use https://marketplace.visualstudio.com/items?itemName=yzhang.markdown-all-in-one -->
- [Installation](#installation)
	- [Download prebuilt binary](#download-prebuilt-binary)
	- [Install using homebrew](#install-using-homebrew)
	- [Install binary from source](#install-binary-from-source)
- [Usage as a CLI tool](#usage-as-a-cli-tool)
	- [Breaking changes](#breaking-changes)
- [Usage as a library](#usage-as-a-library)
- [Usage as a web service](#usage-as-a-web-service)
- [Database migrations](#database-migrations)
- [Included Readers and Writers](#included-readers-and-writers)
- [Development](#development)
	- [Releases](#releases)
- [Licenses](#licenses)

## Installation

### Download prebuilt binary

The `transitland` binaries for Linux and macOS are attached to each [release](https://github.com/interline-io/transitland-lib/releases).

### Install using homebrew

The `transitland` binary can be installed using homebrew. The executable is code-signed and notarized.

```bash
brew install interline-io/transitland-lib/transitland-lib
```

### Install binary from source

```bash
go get github.com/interline-io/transitland-lib/cmd/transitland
```

This package uses Go Modules and will also install required dependencies.

Main dependencies:
- `twpayne/go-geom`
- `jmoiron/sqlx`
- `Masterminds/squirrel`
- `jackc/pgx`
- `mattn/go-sqlite3` (requires CGO)

## Usage as a CLI tool

The main subcommands are:
* [transitland copy](doc/cli/transitland_copy.md)	 - Copy performs a basic copy from a reader to a writer.
* [transitland diff](doc/cli/transitland_diff.md)	 - Calculate difference between two feeds, writing output in a GTFS-like format
* [transitland dmfr-format](doc/cli/transitland_dmfr-format.md)	 - Lint DMFR files
* [transitland dmfr-lint](doc/cli/transitland_dmfr-lint.md)	 - Format a DMFR file
* [transitland extract](doc/cli/transitland_extract.md)	 - Extract a subset of a GTFS feed
* [transitland fetch](doc/cli/transitland_fetch.md)	 - Fetch GTFS data and create feed versions
* [transitland import](doc/cli/transitland_import.md)	 - Import feed versions
* [transitland merge](doc/cli/transitland_merge.md)	 - Merge multiple GTFS feeds
* [transitland sync](doc/cli/transitland_sync.md)	 - Sync DMFR files to database
* [transitland unimport](doc/cli/transitland_unimport.md)	 - Unimport feed versions
* [transitland validate](doc/cli/transitland_validate.md)	 - Validate a GTFS feed
* [transitland rt-convert](doc/cli/transitland_rt-convert.md)	 - Convert GTFS-RealTime to JSON
* [transitland server](doc/cli/transitland_server.md)	 - Run transitland server

See the [full list of subcommands](doc/cli/transitland.md)

### Breaking changes

Note: as of v0.17, we moved from Go standard library `flags` to Cobra's `pflags`; this is a breaking change in that single-dash (`-flag`) command flags are no longer supported, only double-dash (`--flag`).

## Usage as a library

See [library examples](doc/library-example.md).

## Usage as a web service

To start the server with the REST API endpoints, GraphQL API endpoint, GraphQL explorer UI, and image generation endpoints:

```
transitland server --dburl "postgres://your_host/your_database"
```

Alternatively, the database connection string can be specified using `TL_DATABASE_URL` environment variable. For local development environments, you will usually need to add `?sslmode=disable` to the connection string.

Open http://localhost:8080/ in your web browser to see the GraphQL browser, or use the endpoints at `/query` or `/rest/...`

The REST API is documented with OpenAPI 3.0:
- **Interactive documentation**: http://localhost:8080/rest/openapi.json
- **Static schema**: [docs/openapi/rest.json](docs/openapi/rest.json)

The "example" server instance configured by the  `transitland server` command runs without authentication or authorization. Auth configuration is beyond the scope of this example command but can be added by configuring the server in your own package and adding HTTP middlewares to set user context and permissions data. You can use `cmd/tlserver/main.go` as an example to get started; it uses only public APIs from this package. (Earlier versions of `transitland server` included more built-in auth middlewares, but in our experience these are almost always custom per-installation, and were removed from this repo.) Additionally, this example server configuration exposes Go profiler endpoints on `/debug/pprof/...`. 

## Database migrations

Migrations are supported for PostgreSQL, using the schema files in `internal/schema/postgres/migrations`. These files can be read and applied using [golang-migrate](https://github.com/golang-migrate/migrate), which will store the most recently applied migration version in `schema_migrations`. See the `bootstrap.sh` script in that directory for an example, as well as details on how to import Natural Earth data files for associating agencies with places.

SQLite database are intended to be short-lived. They can be created on an as needed basis by passing the `-create` flag to some commands that accept a writer. They use a single executable schema, defined in `internal/schema/sqlite.sql`.

## Included Readers and Writers

| Target                   | Module  | Supports Read | Supports Write |
| ------------------------ | ------- | ------------- | -------------- |
| CSV                      | `tlcsv` | ✅             | ✅              |
| SQLite                   | `tldb`  | ✅             | ✅              |
| PostgreSQL (with PostGIS)  | `tldb`  | ✅             | ✅              |

We welcome the addition of more readers and writers.

## Development

`transitland-lib` follows Go coding conventions.

GitHub Actions runs all tests, stores code coverage reports as artifacts, and prepares releases.

For running tests locally, the following instructions should help get started:

1. Set `TL_TEST_SERVER_DATABASE_URL` to the connection string to a test database
   - e.g. `postgresql://localhost:5432/tlv2_test_server?sslmode=disable`
   - You must also set `PGHOST=localhost`, `PGDATABASE=tlv2_test_server`, etc., to match this url
2. Initialize test fixtures: `./testdata/server/test_setup.sh`
   - This will create the `tlv2_test_server` database in postgres
   - Will halt with an error (intentionally) if this database already exists
   - Runs migrations in `transitland-lib/schema/postgres/migrations`
   - Unpacks and imports the Natural Earth datasets bundled with `transitland-lib`
   - Builds and installs the `cmd/tlserver` command
   - Sets up test feeds contained in `testdata/server/server-test.dmfr.json`
   - Fetches and imports feeds contained in `testdata/server/gtfs`
   - Creates additional fixtures defined in `testdata/server/test_supplement.pgsql`
   - Note that temporary files will be created in `testdata/server/tmp`; these are excluded in `.gitignore`
3. Optional: Set `TL_TEST_REDIS_URL` to run some GBFS tests
4. Optional: Set `TL_TEST_FGA_ENDPOINT` to a running [OpenFGA](https://github.com/openfga/openfga) server to run authorization tests
5. Run all tests with `go test -v ./...`

Test cases generally run within transactions; you do not need to regenerate the fixtures unless you are testing migrations or changes to data import functionality.

### Releases

Releases follow [Semantic Versioning](https://semver.org/) conventions.

To cut a new release:

1. Run `go generate ./...` to update auto-generated documentation.
2. Create a GitHub release. This will create a tag and GitHub Actions will create &amp; attach code-signed binaries.
3. Download the files from the release, and update the [homebrew formula](https://github.com/interline-io/homebrew-transitland-lib/blob/master/transitland-lib.rb) with the updated sha256 hashes and version tag.

## Licenses

`transitland-lib` is released under a "dual license" model:

- open-source for use by all under the [GPLv3](LICENSE) license
- also available under a flexible commercial license from [Interline](mailto:info@interline.io)

