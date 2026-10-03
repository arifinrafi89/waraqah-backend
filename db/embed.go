// Package db embeds the goose migrations so the binary carries its own schema.
package db

import "embed"

// Migrations holds db/migrations/*.sql.
//
//go:embed migrations/*.sql
var Migrations embed.FS
