package store

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type BackupTarget struct {
	ID           uint64    `db:"id"             json:"id"`
	Name         string    `db:"name"           json:"name"`
	Type         string    `db:"type"           json:"type"`
	Bucket       string    `db:"bucket"         json:"bucket"`
	Prefix       string    `db:"prefix"         json:"prefix"`
	AccessKey    string    `db:"access_key"     json:"access_key"`
	SecretKeyEnc string    `db:"secret_key_enc" json:"-"`
	Region       string    `db:"region"         json:"region"`
	Endpoint     string    `db:"endpoint"       json:"endpoint"`
	Enabled      bool      `db:"enabled"        json:"enabled"`
	CreatedAt    time.Time `db:"created_at"     json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"     json:"updated_at"`
}

type BackupTargetStore struct {
	db *sqlx.DB
}

func NewBackupTargetStore(db *sqlx.DB) *BackupTargetStore {
	return &BackupTargetStore{db: db}
}

func (s *BackupTargetStore) List() ([]BackupTarget, error) {
	var targets []BackupTarget
	err := s.db.Select(&targets, "SELECT * FROM backup_targets ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("list backup_targets: %w", err)
	}
	return targets, nil
}

func (s *BackupTargetStore) GetByID(id uint64) (*BackupTarget, error) {
	var t BackupTarget
	if err := s.db.Get(&t, "SELECT * FROM backup_targets WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("get backup_target: %w", err)
	}
	return &t, nil
}

func (s *BackupTargetStore) Create(t *BackupTarget) error {
	q := `INSERT INTO backup_targets (name, type, bucket, prefix, access_key, secret_key_enc, region, endpoint, enabled)
	      VALUES (:name, :type, :bucket, :prefix, :access_key, :secret_key_enc, :region, :endpoint, :enabled)`
	res, err := s.db.NamedExec(q, t)
	if err != nil {
		return fmt.Errorf("insert backup_target: %w", err)
	}
	id, _ := res.LastInsertId()
	t.ID = uint64(id)
	return nil
}

func (s *BackupTargetStore) Update(id uint64, t *BackupTarget) error {
	q := `UPDATE backup_targets SET name=:name, type=:type, bucket=:bucket, prefix=:prefix,
	      access_key=:access_key, secret_key_enc=:secret_key_enc, region=:region,
	      endpoint=:endpoint, enabled=:enabled WHERE id=:id`
	t.ID = id
	_, err := s.db.NamedExec(q, t)
	return err
}

func (s *BackupTargetStore) Delete(id uint64) error {
	_, err := s.db.Exec("DELETE FROM backup_targets WHERE id = ?", id)
	return err
}

func (s *BackupTargetStore) ListEnabled() ([]BackupTarget, error) {
	var targets []BackupTarget
	err := s.db.Select(&targets, "SELECT * FROM backup_targets WHERE enabled = TRUE ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("list enabled backup_targets: %w", err)
	}
	return targets, nil
}
