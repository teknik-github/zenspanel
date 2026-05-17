package store

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

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
