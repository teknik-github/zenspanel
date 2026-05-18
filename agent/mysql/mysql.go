package mysql

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"

	"github.com/zenspanel/zenspanel/agent/safe"
)

type Client struct {
	db *sql.DB
}

func New(dsn string) (*Client, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return &Client{db: db}, nil
}

// CreateDatabase provisions a database and a dedicated user. SQL identifiers
// (db_name, db_user) cannot be parameterized in MySQL, and the password is
// quoted into the CREATE USER statement, so all three inputs must be strictly
// validated here — never relax this without auditing the call sites.
func (c *Client) CreateDatabase(dbName, dbUser, dbPassword string) error {
	if err := safe.DBIdent(dbName); err != nil {
		return err
	}
	if err := safe.DBIdent(dbUser); err != nil {
		return err
	}
	if err := safe.DBPassword(dbPassword); err != nil {
		return err
	}
	queries := []string{
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4", dbName),
		fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'localhost' IDENTIFIED BY '%s'", dbUser, dbPassword),
		fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'localhost'", dbName, dbUser),
		"FLUSH PRIVILEGES",
	}
	for _, q := range queries {
		if _, err := c.db.Exec(q); err != nil {
			return fmt.Errorf("query failed: %w", err)
		}
	}
	return nil
}

func (c *Client) DropDatabase(dbName, dbUser string) error {
	if err := safe.DBIdent(dbName); err != nil {
		return err
	}
	if err := safe.DBIdent(dbUser); err != nil {
		return err
	}
	queries := []string{
		fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", dbName),
		fmt.Sprintf("DROP USER IF EXISTS '%s'@'localhost'", dbUser),
		"FLUSH PRIVILEGES",
	}
	for _, q := range queries {
		if _, err := c.db.Exec(q); err != nil {
			return fmt.Errorf("query failed: %w", err)
		}
	}
	return nil
}
