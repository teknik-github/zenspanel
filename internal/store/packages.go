package store

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type PackageStore struct {
	db *sqlx.DB
}

func NewPackageStore(db *sqlx.DB) *PackageStore {
	return &PackageStore{db: db}
}

func (s *PackageStore) Create(p *Package) error {
	q := `INSERT INTO packages (name, cpu_quota, memory_limit, disk_quota, max_domains, max_databases, php_versions_allowed, terminal_enabled, backup_enabled)
		  VALUES (:name, :cpu_quota, :memory_limit, :disk_quota, :max_domains, :max_databases, :php_versions_allowed, :terminal_enabled, :backup_enabled)`
	res, err := s.db.NamedExec(q, p)
	if err != nil {
		return fmt.Errorf("insert package: %w", err)
	}
	id, _ := res.LastInsertId()
	p.ID = uint64(id)
	return nil
}

func (s *PackageStore) GetByID(id uint64) (*Package, error) {
	var p Package
	if err := s.db.Get(&p, "SELECT * FROM packages WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("get package: %w", err)
	}
	return &p, nil
}

func (s *PackageStore) List() ([]Package, error) {
	var packages []Package
	if err := s.db.Select(&packages, "SELECT * FROM packages ORDER BY name"); err != nil {
		return nil, fmt.Errorf("list packages: %w", err)
	}
	return packages, nil
}

func (s *PackageStore) Update(id uint64, p *Package) error {
	q := `UPDATE packages SET name=:name, cpu_quota=:cpu_quota, memory_limit=:memory_limit,
		  disk_quota=:disk_quota, max_domains=:max_domains, max_databases=:max_databases,
		  max_cron_jobs=:max_cron_jobs, max_procs=:max_procs,
		  io_read_bps=:io_read_bps, io_write_bps=:io_write_bps,
		  antivirus_enabled=:antivirus_enabled, max_ftp_accounts=:max_ftp_accounts,
		  php_versions_allowed=:php_versions_allowed, terminal_enabled=:terminal_enabled,
		  backup_enabled=:backup_enabled WHERE id=:id`
	p.ID = id
	_, err := s.db.NamedExec(q, p)
	return err
}

func (s *PackageStore) Delete(id uint64) error {
	_, err := s.db.Exec("DELETE FROM packages WHERE id = ?", id)
	return err
}
