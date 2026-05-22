package store

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type AdminAllowedIP struct {
	ID        uint64    `db:"id" json:"id"`
	IPCIDR    string    `db:"ip_cidr" json:"ip_cidr"`
	Note      string    `db:"note" json:"note"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type AdminAllowedIPStore struct {
	db *sqlx.DB
}

func NewAdminAllowedIPStore(db *sqlx.DB) *AdminAllowedIPStore {
	return &AdminAllowedIPStore{db: db}
}

func (s *AdminAllowedIPStore) List() ([]AdminAllowedIP, error) {
	var rows []AdminAllowedIP
	if err := s.db.Select(&rows, "SELECT * FROM admin_allowed_ips ORDER BY created_at ASC"); err != nil {
		return nil, fmt.Errorf("list admin_allowed_ips: %w", err)
	}
	return rows, nil
}

func (s *AdminAllowedIPStore) Create(ip, note string) (*AdminAllowedIP, error) {
	_, err := s.db.Exec("INSERT INTO admin_allowed_ips (ip_cidr, note) VALUES (?, ?)", ip, note)
	if err != nil {
		return nil, fmt.Errorf("insert admin_allowed_ip: %w", err)
	}
	var row AdminAllowedIP
	if err := s.db.Get(&row, "SELECT * FROM admin_allowed_ips WHERE ip_cidr = ?", ip); err != nil {
		return nil, fmt.Errorf("get admin_allowed_ip: %w", err)
	}
	return &row, nil
}

func (s *AdminAllowedIPStore) Delete(id uint64) error {
	_, err := s.db.Exec("DELETE FROM admin_allowed_ips WHERE id = ?", id)
	return err
}
