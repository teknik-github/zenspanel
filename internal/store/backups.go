package store

import (
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

var ErrActiveBackup = errors.New("active backup already exists")

type BackupStore struct {
	db *sqlx.DB
}

func NewBackupStore(db *sqlx.DB) *BackupStore {
	return &BackupStore{db: db}
}

func (s *BackupStore) Create(b *Backup) error {
	q := `INSERT INTO backups (user_id, type, status) VALUES (:user_id, :type, :status)`
	res, err := s.db.NamedExec(q, b)
	if err != nil {
		return fmt.Errorf("insert backup: %w", err)
	}
	id, _ := res.LastInsertId()
	b.ID = uint64(id)
	return nil
}

func (s *BackupStore) CreateIfNoActive(b *Backup) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return fmt.Errorf("begin backup tx: %w", err)
	}
	defer tx.Rollback()

	var lockedUserID uint64
	if err := tx.Get(&lockedUserID, "SELECT id FROM users WHERE id = ? FOR UPDATE", b.UserID); err != nil {
		return fmt.Errorf("lock backup owner: %w", err)
	}

	var active int
	if err := tx.Get(
		&active,
		"SELECT COUNT(*) FROM backups WHERE user_id = ? AND status IN ('pending', 'running', 'restoring')",
		b.UserID,
	); err != nil {
		return fmt.Errorf("count active backups: %w", err)
	}
	if active > 0 {
		return ErrActiveBackup
	}

	q := `INSERT INTO backups (user_id, type, status) VALUES (:user_id, :type, :status)`
	res, err := tx.NamedExec(q, b)
	if err != nil {
		return fmt.Errorf("insert backup: %w", err)
	}
	id, _ := res.LastInsertId()
	b.ID = uint64(id)

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit backup tx: %w", err)
	}
	return nil
}

func (s *BackupStore) CountActiveByUserID(userID uint64) (int, error) {
	var active int
	err := s.db.Get(
		&active,
		"SELECT COUNT(*) FROM backups WHERE user_id = ? AND status IN ('pending', 'running', 'restoring')",
		userID,
	)
	return active, err
}

func (s *BackupStore) GetByID(id uint64) (*Backup, error) {
	var b Backup
	if err := s.db.Get(&b, "SELECT * FROM backups WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("get backup: %w", err)
	}
	return &b, nil
}

func (s *BackupStore) ListByUserID(userID uint64) ([]Backup, error) {
	var backups []Backup
	if err := s.db.Select(&backups, "SELECT * FROM backups WHERE user_id = ? ORDER BY created_at DESC", userID); err != nil {
		return nil, fmt.Errorf("list backups: %w", err)
	}
	return backups, nil
}

func (s *BackupStore) UpdateStatus(id uint64, status, filePath string, sizeBytes int64, errMsg string) error {
	_, err := s.db.Exec(
		"UPDATE backups SET status=?, file_path=?, size_bytes=?, error_msg=? WHERE id=?",
		status, filePath, sizeBytes, errMsg, id,
	)
	return err
}

func (s *BackupStore) Delete(id uint64) error {
	_, err := s.db.Exec("DELETE FROM backups WHERE id = ?", id)
	return err
}
