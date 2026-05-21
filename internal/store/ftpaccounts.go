package store

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type FTPAccountStore struct {
	db *sqlx.DB
}

func NewFTPAccountStore(db *sqlx.DB) *FTPAccountStore {
	return &FTPAccountStore{db: db}
}

func (s *FTPAccountStore) Create(a *FTPAccount) error {
	q := `INSERT INTO ftp_accounts (user_id, ftp_username, password_hash, home_dir, enabled)
	      VALUES (:user_id, :ftp_username, :password_hash, :home_dir, :enabled)`
	res, err := s.db.NamedExec(q, a)
	if err != nil {
		return fmt.Errorf("insert ftp_account: %w", err)
	}
	id, _ := res.LastInsertId()
	a.ID = uint64(id)
	return nil
}

func (s *FTPAccountStore) GetByID(id uint64) (*FTPAccount, error) {
	var a FTPAccount
	if err := s.db.Get(&a, "SELECT * FROM ftp_accounts WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("get ftp_account: %w", err)
	}
	return &a, nil
}

func (s *FTPAccountStore) ListByUserID(userID uint64) ([]FTPAccount, error) {
	var accounts []FTPAccount
	if err := s.db.Select(&accounts, "SELECT * FROM ftp_accounts WHERE user_id = ? ORDER BY created_at DESC", userID); err != nil {
		return nil, fmt.Errorf("list ftp_accounts: %w", err)
	}
	return accounts, nil
}

func (s *FTPAccountStore) CountByUserID(userID uint64) (int, error) {
	var count int
	if err := s.db.Get(&count, "SELECT COUNT(*) FROM ftp_accounts WHERE user_id = ?", userID); err != nil {
		return 0, fmt.Errorf("count ftp_accounts: %w", err)
	}
	return count, nil
}

func (s *FTPAccountStore) Delete(id uint64) error {
	_, err := s.db.Exec("DELETE FROM ftp_accounts WHERE id = ?", id)
	return err
}
