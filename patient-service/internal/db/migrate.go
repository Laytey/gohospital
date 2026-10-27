package db

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib" // регистрирует драйвер "pgx" для database/sql
)

// RunMigrations применяет все неприменённые миграции из FS.
//
// Идемпотентна: если миграции уже применены — вернёт nil.
// Внутри открывает отдельное соединение через database/sql,
// потому что golang-migrate не работает с pgxpool напрямую.
func RunMigrations(ctx context.Context, dsn string, fs embed.FS) error {
	log.Println("Накатываем миграции...")

	// 1. Источник миграций — embed.FS, корень "."
	source, err := iofs.New(fs, ".")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// 2. Открываем соединение через database/sql (стандартный интерфейс)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("failed to open db for migrations: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping db for migrations: %w", err)
	}

	// 3. Драйвер migrate для PostgreSQL
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	// 4. Собираем migrate-инстанс
	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("failed to create migrator: %w", err)
	}

	// 5. Применяем все миграции вверх
	if err := m.Up(); err != nil {
		// ErrNoChange — не ошибка, миграции уже применены
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("Миграции уже применены, изменений нет")
			return nil
		}
		return fmt.Errorf("failed to apply migrations: %w", err)
	}

	log.Println("Миграции успешно применены")
	return nil
}
