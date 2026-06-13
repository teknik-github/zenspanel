package store

import (
	"database/sql"
	"fmt"

	mysqldrv "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigrations opens a dedicated connection with multiStatements=true, which
// golang-migrate requires to run multi-statement SQL files. The app connection
// (from store.New) always has multiStatements=false and is never used here.
func RunMigrations(dsn, migrationsPath string) error {
	cfg, err := mysqldrv.ParseDSN(dsn)
	if err != nil {
		return fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MultiStatements = true
	migDB, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return fmt.Errorf("open migration db: %w", err)
	}
	defer migDB.Close()

	driver, err := mysql.WithInstance(migDB, &mysql.Config{})
	if err != nil {
		return fmt.Errorf("migration driver: %w", err)
	}
	m, err := migrate.NewWithDatabaseInstance(
		"file://"+migrationsPath,
		"mysql",
		driver,
	)
	if err != nil {
		return fmt.Errorf("migration init: %w", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migration up: %w", err)
	}
	return nil
}
