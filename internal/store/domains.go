package store

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type DomainStore struct {
	db *sqlx.DB
}

func NewDomainStore(db *sqlx.DB) *DomainStore {
	return &DomainStore{db: db}
}

func (s *DomainStore) Create(d *Domain) error {
	q := `INSERT INTO domains (user_id, domain, document_root, php_version, ssl_type, status)
		  VALUES (:user_id, :domain, :document_root, :php_version, :ssl_type, :status)`
	res, err := s.db.NamedExec(q, d)
	if err != nil {
		return fmt.Errorf("insert domain: %w", err)
	}
	id, _ := res.LastInsertId()
	d.ID = uint64(id)
	return nil
}

func (s *DomainStore) GetByID(id uint64) (*Domain, error) {
	var d Domain
	if err := s.db.Get(&d, "SELECT * FROM domains WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("get domain: %w", err)
	}
	return &d, nil
}

func (s *DomainStore) ListByUserID(userID uint64) ([]Domain, error) {
	var domains []Domain
	if err := s.db.Select(&domains, "SELECT * FROM domains WHERE user_id = ? ORDER BY domain", userID); err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}
	return domains, nil
}

func (s *DomainStore) ListAll() ([]Domain, error) {
	var domains []Domain
	if err := s.db.Select(&domains, "SELECT * FROM domains ORDER BY domain"); err != nil {
		return nil, fmt.Errorf("list all domains: %w", err)
	}
	return domains, nil
}

func (s *DomainStore) CountByUserID(userID uint64) (int, error) {
	var count int
	err := s.db.Get(&count, "SELECT COUNT(*) FROM domains WHERE user_id = ?", userID)
	return count, err
}

func (s *DomainStore) Update(id uint64, fields map[string]interface{}) error {
	fields = filterAllowed(fields, allowedDomainUpdate)
	if len(fields) == 0 {
		return nil
	}
	q := "UPDATE domains SET "
	args := []interface{}{}
	i := 0
	for k, v := range fields {
		if i > 0 {
			q += ", "
		}
		q += k + " = ?"
		args = append(args, v)
		i++
	}
	q += " WHERE id = ?"
	args = append(args, id)
	_, err := s.db.Exec(q, args...)
	return err
}

func (s *DomainStore) Delete(id uint64) error {
	_, err := s.db.Exec("DELETE FROM domains WHERE id = ?", id)
	return err
}

func (s *DomainStore) GetByDomain(domain string) (*Domain, error) {
	var d Domain
	if err := s.db.Get(&d, "SELECT * FROM domains WHERE domain = ?", domain); err != nil {
		return nil, fmt.Errorf("get domain by name: %w", err)
	}
	return &d, nil
}
