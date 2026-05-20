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
		// Revoke any global privileges before granting — defense in depth (V41).
		// If the user already existed with broader grants, this strips them.
		fmt.Sprintf("REVOKE ALL PRIVILEGES, GRANT OPTION FROM '%s'@'localhost'", dbUser),
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

// GetUserDBSize returns the total size in bytes of all databases owned
// by dbUser (matched by db_name LIKE '<dbUser>_%'). Uses information_schema
// so no filesystem access is needed and it works regardless of MySQL data dir.
func (c *Client) GetUserDBSize(dbUser string) (int64, error) {
	if err := safe.DBIdent(dbUser); err != nil {
		return 0, err
	}
	// Match all databases belonging to this user: convention is <user>_<dbname>
	pattern := dbUser + "_%"
	var size int64
	err := c.db.QueryRow(`
		SELECT COALESCE(SUM(data_length + index_length), 0)
		FROM information_schema.tables
		WHERE table_schema LIKE ?`, pattern).Scan(&size)
	if err != nil {
		return 0, fmt.Errorf("get db size: %w", err)
	}
	return size, nil
}

// EnforceDBQuota checks if the user's total DB size exceeds hardBytes.
// If over quota: revokes INSERT, CREATE, UPDATE on all user DBs.
// If under quota: restores those privileges.
// hardBytes=0 means unlimited — no enforcement.
func (c *Client) EnforceDBQuota(dbUser string, hardBytes int64) error {
	if err := safe.DBIdent(dbUser); err != nil {
		return err
	}
	if hardBytes <= 0 {
		return nil // unlimited
	}

	used, err := c.GetUserDBSize(dbUser)
	if err != nil {
		return err
	}

	pattern := dbUser + "_%"
	// Get list of databases for this user.
	rows, err := c.db.Query(
		"SELECT schema_name FROM information_schema.schemata WHERE schema_name LIKE ?", pattern)
	if err != nil {
		return fmt.Errorf("list user dbs: %w", err)
	}
	defer rows.Close()

	var dbs []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			dbs = append(dbs, name)
		}
	}

	for _, db := range dbs {
		if used >= hardBytes {
			// Over quota — revoke write privileges.
			_, _ = c.db.Exec(fmt.Sprintf(
				"REVOKE INSERT, CREATE, UPDATE ON `%s`.* FROM '%s'@'localhost'",
				db, dbUser))
		} else {
			// Under quota — restore write privileges.
			_, _ = c.db.Exec(fmt.Sprintf(
				"GRANT INSERT, CREATE, UPDATE ON `%s`.* TO '%s'@'localhost'",
				db, dbUser))
		}
	}
	_, _ = c.db.Exec("FLUSH PRIVILEGES")
	return nil
}

// ResetUserPassword changes a panel-managed MySQL user's password to a new
// value, used by the phpMyAdmin SSO flow: we mint a one-shot password,
// hand it to phpMyAdmin's auto-submit form, and discard it. The original
// password is gone — we never stored it — so a new one is the only way
// to log the user in non-interactively.
//
// Same identifier-and-password validation as CreateDatabase: dbUser must
// be alphanumeric/underscore, password must come from the safe charset.
// MySQL has no parameterised form for ALTER USER's IDENTIFIED BY, so this
// gate is the only thing keeping injection out.
func (c *Client) ResetUserPassword(dbUser, newPassword string) error {
	if err := safe.DBIdent(dbUser); err != nil {
		return err
	}
	if err := safe.DBPassword(newPassword); err != nil {
		return err
	}
	queries := []string{
		fmt.Sprintf("ALTER USER '%s'@'localhost' IDENTIFIED BY '%s'", dbUser, newPassword),
		"FLUSH PRIVILEGES",
	}
	for _, q := range queries {
		if _, err := c.db.Exec(q); err != nil {
			return fmt.Errorf("query failed: %w", err)
		}
	}
	return nil
}
