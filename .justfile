set dotenv-load := true

# System user for PostgreSQL defaults (handy for macOS/Homebrew)
USER := `whoami`

export TL_DATABASE_URL := env_var_or_default("TL_DATABASE_URL", "postgres://" + USER + "@localhost:5432/transitland?sslmode=disable")
export TL_TEST_DATABASE_URL := env_var_or_default("TL_TEST_DATABASE_URL", "postgres://" + USER + "@localhost:5432/tlv2_test?sslmode=disable")
export TL_TEST_SERVER_DATABASE_URL := env_var_or_default("TL_TEST_SERVER_DATABASE_URL", "postgres://" + USER + "@localhost:5432/tlv2_test_server?sslmode=disable")
export TL_REDIS_URL := env_var_or_default("TL_REDIS_URL", "redis://localhost")
export TL_GBFS_VERSION := env_var_or_default("TL_GBFS_VERSION", "2.3")

# List all recipes
default:
    @just --list

# Build and install the transitland command (preferred way to update binary)
build:
    cd cmd/transitland && go install .

# Run the transitland command from source
run *args:
    cd cmd/transitland && go run . {{args}}


# Run all formatting, linting, and tidying tasks
tidy:
    go generate ./...
    go fmt ./...
    go vet ./... # Perhaps in the future replace with golangci-lint
    go mod tidy

# --- Testing ---

# Run all tests
test *args:
    go test -v {{args}} ./...

# Run tests and generate coverage report
coverage:
    go test -v -coverprofile c.out ./...
    go tool cover -html=c.out -o coverage.html
    @echo "Coverage report generated at coverage.html"

# Initialize test database and fixtures (requires built binary)
test-setup: build
    ./testdata/test_setup.sh

# Complete test run: build, setup fixtures, and run tests
test-all: test-setup
    just test

# --- Database ---

# Run database migrations (requires built binary)
db-migrate command: build
    transitland dbmigrate --dburl="{{TL_DATABASE_URL}}" {{command}}

# Bootstrap a postgres database with Natural Earth data (requires built binary)
db-bootstrap *args: build
    ./schema/postgres/bootstrap.sh {{args}}

# Create a new migration file
db-create-migration name:
    migrate create -ext=.pgsql -dir=schema/postgres/migrations {{name}}

# --- Specification ---

# Update local copy of GTFS specification
spec-update-gtfs:
    wget https://github.com/google/transit/raw/refs/heads/master/gtfs/spec/en/reference.md -O gtfs/reference.md
    @HASH=$(git ls-remote https://github.com/google/transit.git master | awk '{print $1}'); \
    sed -i '' "s/var GTFSVERSION = \".*\"/var GTFSVERSION = \"$HASH\"/" version.go

# Update local copy of GBFS human-readable spec
spec-update-gbfs:
    wget https://github.com/MobilityData/gbfs/raw/refs/heads/master/gbfs.md -O internal/gbfs/gbfs.md

# Update local copy of GBFS JSON schema (defaults to TL_GBFS_VERSION)
spec-update-gbfs-schema version=TL_GBFS_VERSION:
    @mkdir -p internal/gbfs/schema/v{{version}}
    wget https://raw.githubusercontent.com/MobilityData/gbfs-json-schema/master/v{{version}}/gbfs.json -O internal/gbfs/schema/v{{version}}/gbfs.json
    @sed -i '' "s/var GBFS_SCHEMA_VERSION = \".*\"/var GBFS_SCHEMA_VERSION = \"v{{version}}\"/" version.go

# Update GTFS-Realtime protobuf definition and recompile
spec-update-gtfs-rt:
    wget https://github.com/google/transit/raw/refs/heads/master/gtfs-realtime/proto/gtfs-realtime.proto -O rt/pb/gtfs-realtime.proto
    cd rt/pb && protoc --plugin=protoc-gen-go=../../protoc-gen-go-wrapper.sh --go_out=. --go_opt=paths=source_relative --go_opt=Mgtfs-realtime.proto=rt/pb gtfs-realtime.proto
    @HASH=$(git ls-remote https://github.com/google/transit.git master | awk '{print $1}'); \
    sed -i '' "s/var GTFSRTVERSION = \".*\"/var GTFSRTVERSION = \"$HASH\"/" version.go