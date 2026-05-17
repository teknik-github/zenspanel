package mysql

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
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

func (c *Client) CreateDatabase(dbName, dbUser, dbPassword string) error {
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
