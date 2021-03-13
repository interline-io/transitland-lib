package schema

import _ "embed"

//go:embed postgres.pgsql
var PostgresSchema string

//go:embed sqlite.sql
var SqliteSchema string
