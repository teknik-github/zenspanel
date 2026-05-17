package store

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type PHPVersionStore struct {
	db *sqlx.DB
}

func NewPHPVersionStore(db *sqlx.DB) *PHPVersionStore {
	return &PHPVersionStore{db: db}
}

func (s *PHPVersionStore) List() ([]PHPVersion, error) {
	var versions []PHPVersion
	if err := s.db.Select(&versions, "SELECT * FROM php_versions ORDER BY version DESC"); err != nil {
		return nil, fmt.Errorf("list php versions: %w", err)
	}
	return versions, nil
}

func (s *PHPVersionStore) ListEnabled() ([]PHPVersion, error) {
	var versions []PHPVersion
	if err := s.db.Select(&versions, "SELECT * FROM php_versions WHERE enabled = TRUE ORDER BY version DESC"); err != nil {
		return nil, fmt.Errorf("list enabled php versions: %w", err)
	}
	return versions, nil
}

func (s *PHPVersionStore) GetByID(id uint64) (*PHPVersion, error) {
	var v PHPVersion
	if err := s.db.Get(&v, "SELECT * FROM php_versions WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("get php version: %w", err)
	}
	return &v, nil
}

func (s *PHPVersionStore) SetEnabled(id uint64, enabled bool) error {
	_, err := s.db.Exec("UPDATE php_versions SET enabled = ? WHERE id = ?", enabled, id)
	return err
}
