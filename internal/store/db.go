package store

import (
	"fmt"

	mysqldrv "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func New(dsn string) (*sqlx.DB, error) {
	// Parse and enforce multiStatements=false regardless of what the operator
	// put in config.yaml. That flag exists only on the migration connection.
	cfg, err := mysqldrv.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MultiStatements = false
	db, err := sqlx.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	return db, nil
}
