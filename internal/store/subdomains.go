package store

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type SubdomainStore struct {
	db *sqlx.DB
}

func NewSubdomainStore(db *sqlx.DB) *SubdomainStore {
	return &SubdomainStore{db: db}
}

func (s *SubdomainStore) Create(d *Subdomain) error {
	q := `INSERT INTO subdomains
		(user_id, parent_domain_id, subdomain, fqdn, document_root, php_version, ssl_type, status)
		VALUES
		(:user_id, :parent_domain_id, :subdomain, :fqdn, :document_root, :php_version, :ssl_type, :status)`
	res, err := s.db.NamedExec(q, d)
	if err != nil {
		return fmt.Errorf("insert subdomain: %w", err)
	}
	id, _ := res.LastInsertId()
	d.ID = uint64(id)
	return nil
}

func (s *SubdomainStore) GetByID(id uint64) (*Subdomain, error) {
	var d Subdomain
	if err := s.db.Get(&d, "SELECT * FROM subdomains WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("get subdomain: %w", err)
	}
	return &d, nil
}

// GetByFQDN powers the collision check at create time so a subdomain
// can't shadow an existing parent domain or another user's subdomain.
// Returns (nil, nil) when no row matches — the caller treats that as
// "free to use".
func (s *SubdomainStore) GetByFQDN(fqdn string) (*Subdomain, error) {
	var d Subdomain
	err := s.db.Get(&d, "SELECT * FROM subdomains WHERE fqdn = ?", fqdn)
	if err != nil {
		// sqlx returns sql.ErrNoRows wrapped — caller wants a nil
		// row rather than wrestling with the error type, so we just
		// flatten that into (nil, nil).
		return nil, nil
	}
	return &d, nil
}

func (s *SubdomainStore) ListByUserID(userID uint64) ([]Subdomain, error) {
	var subs []Subdomain
	if err := s.db.Select(&subs,
		"SELECT * FROM subdomains WHERE user_id = ? ORDER BY fqdn", userID); err != nil {
		return nil, fmt.Errorf("list subdomains by user: %w", err)
	}
	return subs, nil
}

func (s *SubdomainStore) ListByParentID(parentID uint64) ([]Subdomain, error) {
	var subs []Subdomain
	if err := s.db.Select(&subs,
		"SELECT * FROM subdomains WHERE parent_domain_id = ? ORDER BY subdomain", parentID); err != nil {
		return nil, fmt.Errorf("list subdomains by parent: %w", err)
	}
	return subs, nil
}

func (s *SubdomainStore) ListAll() ([]Subdomain, error) {
	var subs []Subdomain
	if err := s.db.Select(&subs, "SELECT * FROM subdomains ORDER BY fqdn"); err != nil {
		return nil, fmt.Errorf("list all subdomains: %w", err)
	}
	return subs, nil
}

func (s *SubdomainStore) CountByUserID(userID uint64) (int, error) {
	var count int
	err := s.db.Get(&count, "SELECT COUNT(*) FROM subdomains WHERE user_id = ?", userID)
	return count, err
}

func (s *SubdomainStore) Update(id uint64, fields map[string]interface{}) error {
	fields = filterAllowed(fields, allowedSubdomainUpdate)
	if len(fields) == 0 {
		return nil
	}
	q := "UPDATE subdomains SET "
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

func (s *SubdomainStore) Delete(id uint64) error {
	_, err := s.db.Exec("DELETE FROM subdomains WHERE id = ?", id)
	return err
}
