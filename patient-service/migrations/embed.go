// Package migrations содержит SQL-миграции для patient_db.
// Файлы встраиваются в бинарник через //go:embed.
package migrations

import "embed"

// FS — виртуальная файловая система с миграциями.
// В неё попадают все .sql файлы из текущей директории.
//
//go:embed *.sql
var FS embed.FS
