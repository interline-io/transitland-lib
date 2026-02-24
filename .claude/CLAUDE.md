# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`transitland-lib` is a Go library and CLI tool for reading, writing, and processing transit data in GTFS and related formats. It provides CSV/database readers and writers, a transformation pipeline (copier), GTFS validation, a GraphQL/REST web API, and DMFR (Distributed Mobility Feed Registry) support.

**Go 1.24.2** | PostgreSQL/PostGIS | SQLite (requires CGO) | GraphQL (gqlgen) | Cobra CLI

## Build & Test Commands

```bash
# Install CLI binary
(cd cmd/transitland && go install .)

# Run all tests
go test -v ./...

# Run tests for a specific package
go test -v ./tlcsv/...

# Run a single test
go test -run TestReaderStructure -v ./tlcsv/...

# Run with coverage
go test -v -coverprofile c.out ./...

# Regenerate GraphQL code after schema changes
(cd internal/generated/gqlout && go generate)

# Regenerate all auto-generated code (docs, etc.)
go generate ./...
```

### Test Database Setup

Many tests (especially in `server/`, `tldb/`, `importer/`) require PostgreSQL with PostGIS. Set these environment variables and run the setup script:

```bash
export TL_TEST_DATABASE_URL="postgres://root:for_testing@localhost:5432/tlv2_test?sslmode=disable"
export TL_TEST_SERVER_DATABASE_URL="postgres://root:for_testing@localhost:5432/tlv2_test_server?sslmode=disable"
./testdata/test_setup.sh
```

Tests run within transactions; fixtures only need regenerating when testing migrations or import changes. Optional: `TL_TEST_REDIS_URL` for GBFS tests, `TL_TEST_FGA_ENDPOINT` for authorization tests.

## Architecture

### Core Data Flow: Reader → Copier → Writer

The central abstraction is the **adapter pattern** in `adapters/`:
- **Reader** (`adapters.Reader`): streams GTFS entities via channels. Implementations: `tlcsv` (ZIP/directory/HTTP), `tldb` (PostgreSQL/SQLite)
- **Writer** (`adapters.Writer`): accepts entities via `AddEntity(tt.Entity)`. Same implementations as Reader.
- **Copier** (`copier/`): transformation pipeline that connects Reader to Writer through a chain of `Filter → Validate → AfterValidator → Extension → Write`

### Entity Type System (`tt/` and `gtfs/`)

GTFS fields use nullable `Option[T]` wrappers defined in `tt/`:
```go
type Option[T any] struct { Val T; Valid bool }
```

All GTFS entities (in `gtfs/`) implement `tt.Entity` with `EntityID()` and `Filename()`. Type aliases like `tt.String`, `tt.Int` are `Option[string]`, `Option[int]`, etc.

### Key Packages

| Package | Role |
|---------|------|
| `gtfs/` | GTFS entity struct definitions |
| `tt/` | Nullable Option types, Entity interface, EntityMap for ID mapping |
| `adapters/` | Reader/Writer interfaces |
| `tlcsv/` | CSV/ZIP reader and writer |
| `tldb/` | Database adapter; `tldb/postgres/` and `tldb/sqlite/` register drivers |
| `copier/` | Transformation pipeline with Filter, Validator, Marker, Extension interfaces |
| `ext/` | Copier extensions: `ext/filters/` (transforms), `ext/builders/` (geometry, onestop IDs), `ext/plus/` (extra entity types) |
| `validator/` | GTFS feed validation engine |
| `importer/` | Database import with feed version tracking |
| `server/` | Web server: GraphQL + REST APIs |
| `server/gql/` | GraphQL resolvers and data loaders |
| `server/finders/dbfinder/` | Database query implementations for GraphQL |
| `server/model/` | GraphQL model types and Finder interfaces |
| `dmfr/` | DMFR feed registry support |
| `service/` | Calendar + CalendarDate → Service abstraction |
| `tlxy/` | Geospatial utilities |
| `rt/` | GTFS-RealTime support |
| `cmds/` | CLI command implementations |
| `cmd/transitland/` | CLI entry point |
| `causes/` | Error context tracking (file, line, entity, field) |

### Database Schema

- PostgreSQL migrations: `schema/postgres/migrations/*.up.pgsql` (applied via `transitland dbmigrate up`)
- SQLite schema: `schema/sqlite/sqlite.sql` (single-file, created on demand with `-create` flag)

### GraphQL API

- Schema files: `schema/graphql/*.graphqls`
- Code generation config: `gqlgen.yml` → generates into `internal/generated/gqlout/`
- Generated filter types: `server/model/generated_models.go`
- Resolvers use data loaders (`server/gql/loaders.go`) for batched database access
- See `server/gql/RESOLVER_GUIDE.md` for the complete step-by-step process of adding a new entity/resolver

### Copier Extension Interfaces

Extensions plug into the copier pipeline at different stages:
- `Prepare`: runs before copying begins
- `Filter`: transforms entities before validation
- `Marker`/`EntityMarker`: selects which entities to process
- `ExpandFilter`: duplicates/expands entities
- `Validator`: validates individual entities
- `AfterValidator`: post-validation hooks
- `AfterWrite`: post-write hooks
- `Extension`: custom batch processing after normal copying

### Driver Registration Pattern

Database drivers and extensions use Go `init()` side-effect imports. The CLI entry point (`cmd/transitland/main.go`) imports these with blank identifiers:
```go
_ "github.com/interline-io/transitland-lib/tldb/postgres"
_ "github.com/interline-io/transitland-lib/tldb/sqlite"
_ "github.com/interline-io/transitland-lib/ext/filters"
```

### GTFS Specification References

This library tracks specific commits of the upstream GTFS specs, defined in `version.go`:
- **GTFS Static**: `GTFSVERSION` → commit hash pointing to `https://github.com/google/transit/blob/{hash}/gtfs/spec/en/reference.md`
- **GTFS Realtime**: `GTFSRTVERSION` → commit hash pointing to `https://github.com/google/transit/blob/{hash}/gtfs-realtime/proto/gtfs-realtime.proto`
- **GTFS Realtime proto**: bundled at `rt/pb/gtfs-realtime.proto` with generated Go code in `rt/pb/gtfs-realtime.pb.go`
- **Extended route types**: defined in `tt/routetypes.go`, based on https://developers.google.com/transit/gtfs/reference/extended-route-types
- **GTFS-RT validation rules**: `rt/errors.go` maps 40+ error codes from https://github.com/CUTR-at-USF/gtfs-realtime-validator/blob/master/RULES.md
- **Spec-based validation**: `rules/` contains validation rules referencing specific GTFS spec requirements (e.g., stop location_type constraints for transfers, stop_time sequence rules)

Entity structs in `gtfs/` have filenames matching their GTFS source files (e.g., `agency.go` for `agency.txt`). GraphQL type descriptions link to the relevant spec section (e.g., `https://gtfs.org/reference/static/#agencytxt`).

### Testing Patterns

- Uses `testify/assert` for assertions
- CSV reader tests use `internal/testreader.ReaderTester` with expected entity counts
- GraphQL resolver tests use `testcase` structs with GraphQL queries and `queryTestcase()` helper
- Test data lives in `testdata/` (GTFS fixtures, server test data, DMFR files)
