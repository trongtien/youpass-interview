package database

import (
	"errors"
	"fmt"
	"io/fs"
	"log"

	"github.com/golang-migrate/migrate/v4"
	pgxmigrate "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// RunMigrations applies every pending up migration from source.
// golang-migrate holds a Postgres advisory lock while running, so several
// instances starting at once apply each migration only once.
func RunMigrations(pool *pgxpool.Pool, source fs.FS) error {
	src, err := iofs.New(source, ".")
	if err != nil {
		return fmt.Errorf("Open migration source failed %w", err)
	}

	// Closing this *sql.DB does not close the pool.
	db := stdlib.OpenDBFromPool(pool)
	driver, err := pgxmigrate.WithInstance(db, &pgxmigrate.Config{})
	if err != nil {
		src.Close()
		db.Close()
		return fmt.Errorf("Open migration driver failed %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "pgx5", driver)
	if err != nil {
		src.Close()
		driver.Close()
		return fmt.Errorf("Init migration failed %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("Apply migration failed %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return fmt.Errorf("Read migration version failed %w", err)
	}
	log.Printf("Database migrated to version %d (dirty=%t)", version, dirty)

	return nil
}
