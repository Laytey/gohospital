// Package migrations содержит SQL-миграции для appointments_db.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
