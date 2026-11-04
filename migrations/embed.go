// Package migrations embeds the database migrations in the binary.
package migrations

import "embed"

// FS contains the numbered migration files.
//
//go:embed *.sql
var FS embed.FS
