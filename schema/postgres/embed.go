package postgres

import "embed"

//go:embed migrations/*.pgsql
var EmbeddedMigrations embed.FS
