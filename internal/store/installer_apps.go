package store

import "github.com/jmoiron/sqlx"

type InstallerAppStore struct{ db *sqlx.DB }

// InstallerAppSetting records the admin-controlled enabled state for one app slug.
type InstallerAppSetting struct {
	Slug    string `db:"slug"    json:"slug"`
	Enabled bool   `db:"enabled" json:"enabled"`
}

func NewInstallerAppStore(db *sqlx.DB) *InstallerAppStore {
	return &InstallerAppStore{db: db}
}

func (s *InstallerAppStore) List() ([]InstallerAppSetting, error) {
	var rows []InstallerAppSetting
	return rows, s.db.Select(&rows, "SELECT slug, enabled FROM installer_apps ORDER BY slug")
}

func (s *InstallerAppStore) SetEnabled(slug string, enabled bool) error {
	_, err := s.db.Exec("UPDATE installer_apps SET enabled = ? WHERE slug = ?", enabled, slug)
	return err
}

// EnabledMap returns a set of slugs that are currently enabled.
func (s *InstallerAppStore) EnabledMap() (map[string]bool, error) {
	rows, err := s.List()
	if err != nil {
		return nil, err
	}
	m := make(map[string]bool, len(rows))
	for _, r := range rows {
		m[r.Slug] = r.Enabled
	}
	return m, nil
}
