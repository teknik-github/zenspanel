package store

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type PHPExtension struct {
	ID         uint64    `db:"id"          json:"id"`
	Name       string    `db:"name"        json:"name"`
	PHPVersion string    `db:"php_version" json:"php_version"`
	Enabled    bool      `db:"enabled"     json:"enabled"`
	CreatedAt  time.Time `db:"created_at"  json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"  json:"updated_at"`
}

type UserPHPExtension struct {
	ID        uint64    `db:"id"         json:"id"`
	UserID    uint64    `db:"user_id"    json:"user_id"`
	ExtID     uint64    `db:"ext_id"     json:"ext_id"`
	Enabled   bool      `db:"enabled"    json:"enabled"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

// UserExtView is the merged view returned to the user: global default
// plus any per-user override. admin_enabled=false means the user cannot
// re-enable regardless of their own setting (V20).
type UserExtView struct {
	ID           uint64 `db:"id"           json:"id"`
	Name         string `db:"name"         json:"name"`
	PHPVersion   string `db:"php_version"  json:"php_version"`
	AdminEnabled bool   `db:"admin_enabled" json:"admin_enabled"`
	UserEnabled  bool   `db:"user_enabled"  json:"user_enabled"`
}

type PHPExtensionStore struct {
	db *sqlx.DB
}

func NewPHPExtensionStore(db *sqlx.DB) *PHPExtensionStore {
	return &PHPExtensionStore{db: db}
}

// Create inserts a new extension into the global catalog.
// Returns an error if the (name, php_version) pair already exists.
func (s *PHPExtensionStore) Create(ext *PHPExtension) error {
	q := `INSERT IGNORE INTO php_extensions (name, php_version, enabled)
	      VALUES (?, ?, ?)`
	res, err := s.db.Exec(q, ext.Name, ext.PHPVersion, ext.Enabled)
	if err != nil {
		return fmt.Errorf("insert php_extension: %w", err)
	}
	id, _ := res.LastInsertId()
	if id == 0 {
		return fmt.Errorf("already exists")
	}
	ext.ID = uint64(id)
	return nil
}

// List returns all rows in the global catalog, optionally filtered by
// php_version. Used by the admin page.
func (s *PHPExtensionStore) List(phpVersion string) ([]PHPExtension, error) {
	var exts []PHPExtension
	var err error
	if phpVersion != "" {
		err = s.db.Select(&exts,
			"SELECT * FROM php_extensions WHERE php_version = ? ORDER BY name",
			phpVersion)
	} else {
		err = s.db.Select(&exts,
			"SELECT * FROM php_extensions ORDER BY php_version, name")
	}
	if err != nil {
		return nil, fmt.Errorf("list php_extensions: %w", err)
	}
	return exts, nil
}

// GetByID returns a single global extension row.
func (s *PHPExtensionStore) GetByID(id uint64) (*PHPExtension, error) {
	var ext PHPExtension
	if err := s.db.Get(&ext, "SELECT * FROM php_extensions WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("get php_extension: %w", err)
	}
	return &ext, nil
}

// SetGlobal toggles the admin-level enabled flag for an extension.
// When an admin disables an extension, users who had it enabled will
// effectively have it disabled too (V20 — enforced at read time in
// GetUserState and at write time in SetUserState).
func (s *PHPExtensionStore) SetGlobal(id uint64, enabled bool) error {
	_, err := s.db.Exec(
		"UPDATE php_extensions SET enabled = ? WHERE id = ?", enabled, id)
	return err
}

// GetUserState returns the merged extension list for a user: global
// catalog joined with any per-user overrides. user_enabled reflects the
// effective state — if admin disabled the ext, user_enabled is forced
// false regardless of the override row (V20).
func (s *PHPExtensionStore) GetUserState(userID uint64, phpVersion string) ([]UserExtView, error) {
	query := `
		SELECT
			pe.id,
			pe.name,
			pe.php_version,
			pe.enabled                                          AS admin_enabled,
			CASE
				WHEN pe.enabled = FALSE THEN FALSE
				WHEN upe.enabled IS NOT NULL THEN upe.enabled
				ELSE pe.enabled
			END                                                 AS user_enabled
		FROM php_extensions pe
		LEFT JOIN user_php_extensions upe
			ON upe.ext_id = pe.id AND upe.user_id = ?
		WHERE pe.php_version = ?
		ORDER BY pe.name`
	var views []UserExtView
	if err := s.db.Select(&views, query, userID, phpVersion); err != nil {
		return nil, fmt.Errorf("get user ext state: %w", err)
	}
	return views, nil
}

// GetUsersWithExtEnabled returns userIDs + usernames of users who have
// the given extension explicitly enabled (user override = true). Used by
// AdminUpdate to propagate a global disable to running FPM pools (V44).
func (s *PHPExtensionStore) GetUsersWithExtEnabled(extID uint64) ([]struct {
	UserID   uint64
	Username string
}, error) {
	rows, err := s.db.Query(`
		SELECT u.id, u.username
		FROM user_php_extensions upe
		JOIN users u ON u.id = upe.user_id
		WHERE upe.ext_id = ? AND upe.enabled = TRUE`, extID)
	if err != nil {
		return nil, fmt.Errorf("get users with ext enabled: %w", err)
	}
	defer rows.Close()
	var result []struct {
		UserID   uint64
		Username string
	}
	for rows.Next() {
		var r struct {
			UserID   uint64
			Username string
		}
		if err := rows.Scan(&r.UserID, &r.Username); err != nil {
			continue
		}
		result = append(result, r)
	}
	return result, nil
}

// SetUserState upserts a per-user override. Returns an error if the
// admin has disabled the extension and the user is trying to enable it
// (V20). Callers should check this before calling the agent.
func (s *PHPExtensionStore) SetUserState(userID, extID uint64, enabled bool) error {
	// Enforce V20: cannot enable an admin-disabled extension.
	if enabled {
		var adminEnabled bool
		if err := s.db.Get(&adminEnabled,
			"SELECT enabled FROM php_extensions WHERE id = ?", extID); err != nil {
			return fmt.Errorf("lookup ext: %w", err)
		}
		if !adminEnabled {
			return fmt.Errorf("extension is disabled by admin")
		}
	}
	_, err := s.db.Exec(`
		INSERT INTO user_php_extensions (user_id, ext_id, enabled)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE enabled = VALUES(enabled)`,
		userID, extID, enabled)
	return err
}

func (s *PHPExtensionStore) Delete(id uint64) error {
	_, err := s.db.Exec("DELETE FROM php_extensions WHERE id = ?", id)
	return err
}
