package store

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type DomainRedirect struct {
	ID         uint64    `db:"id"          json:"id"`
	DomainID   uint64    `db:"domain_id"   json:"domain_id"`
	SourcePath string    `db:"source_path" json:"source_path"`
	DestURL    string    `db:"dest_url"    json:"dest_url"`
	Type       string    `db:"type"        json:"type"`
	Enabled    bool      `db:"enabled"     json:"enabled"`
	CreatedAt  time.Time `db:"created_at"  json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"  json:"updated_at"`
}

type RedirectStore struct {
	db *sqlx.DB
}

func NewRedirectStore(db *sqlx.DB) *RedirectStore {
	return &RedirectStore{db: db}
}

func (s *RedirectStore) ListByDomainID(domainID uint64) ([]DomainRedirect, error) {
	var rows []DomainRedirect
	err := s.db.Select(&rows,
		"SELECT * FROM domain_redirects WHERE domain_id = ? ORDER BY id", domainID)
	if err != nil {
		return nil, fmt.Errorf("list redirects: %w", err)
	}
	return rows, nil
}

func (s *RedirectStore) GetByID(id uint64) (*DomainRedirect, error) {
	var r DomainRedirect
	if err := s.db.Get(&r, "SELECT * FROM domain_redirects WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("get redirect: %w", err)
	}
	return &r, nil
}

func (s *RedirectStore) Create(r *DomainRedirect) error {
	q := `INSERT INTO domain_redirects (domain_id, source_path, dest_url, type, enabled)
	      VALUES (:domain_id, :source_path, :dest_url, :type, :enabled)`
	res, err := s.db.NamedExec(q, r)
	if err != nil {
		return fmt.Errorf("insert redirect: %w", err)
	}
	id, _ := res.LastInsertId()
	r.ID = uint64(id)
	return nil
}

func (s *RedirectStore) Update(id uint64, r *DomainRedirect) error {
	q := `UPDATE domain_redirects SET source_path=:source_path, dest_url=:dest_url,
	      type=:type, enabled=:enabled WHERE id=:id`
	r.ID = id
	_, err := s.db.NamedExec(q, r)
	return err
}

func (s *RedirectStore) Delete(id uint64) error {
	_, err := s.db.Exec("DELETE FROM domain_redirects WHERE id = ?", id)
	return err
}
