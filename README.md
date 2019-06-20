# Interline GoTransit

building blocks for processing transit data. can be used either on the command line or within wrappers/programs

including GTFS, GTFS-Realtime, future potential for GBFS, MDS, etc, etc

uses [DMFR](https://github.com/transitland/distributed-mobility-feed-registry) format to specify multiple input feeds

## Concepts

- reader
- copier
- filter
- marker
- writer

## Installation

Linux binaries are attached to each [release](https://github.com/interline-io/gotransit/releases).

To use on Mac or Windows, compile locally like so:

```bash
# TODO: example
```

Optional dependencies:

- sqlite
- spatialite
- postgres/postgis

## Usage as a CLI tool

### copy command

`gotransit copy`

### extract command

`gotransit extract` 

### set command

`gotransit set`

### validate command

`gotransit validate`

## Development

GoTransit follows Golang coding conventions.

CircleCI runs all tests and stores code coverage reports as artifacts at https://circleci.com/gh/interline-io/gotransit

### Releases

Releases follow [Semantic Versioning](https://semver.org/) conventions.

To cut a new release:

1. Tag the `master` branch with the next SemVer version (for example: `v0.2.0`).
2. CircleCI will run [GoReleaser](https://github.com/goreleaser/goreleaser) and create a GitHub release on this repository.

## Licenses

GoTransit is released under a "dual license" model:

- open-source for use by all under the GPLv3 license
- also available under a flexible commercial license from Interline