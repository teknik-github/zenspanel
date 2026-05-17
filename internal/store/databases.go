package store

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type DatabaseStore struct {
	db *sqlx.DB
}

func NewDatabaseStore(db *sqlx.DB) *DatabaseStore {
	return &DatabaseStore{db: db}
}

func (s *DatabaseStore) Create(d *Database) error {
	q := `INSERT INTO ` + "`databases`" + ` (user_id, db_name, db_user) VALUES (:user_id, :db_name, :db_user)`
	res, err := s.db.NamedExec(q, d)
	if err != nil {
		return fmt.Errorf("insert database: %w", err)
	}
	id, _ := res.LastInsertId()
	d.ID = uint64(id)
	return nil
}

func (s *DatabaseStore) GetByID(id uint64) (*Database, error) {
	var d Database
	if err := s.db.Get(&d, "SELECT * FROM `databases` WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("get database: %w", err)
	}
	return &d, nil
}

func (s *DatabaseStore) ListByUserID(userID uint64) ([]Database, error) {
	var dbs []Database
	if err := s.db.Select(&dbs, "SELECT * FROM `databases` WHERE user_id = ? ORDER BY db_name", userID); err != nil {
		return nil, fmt.Errorf("list databases: %w", err)
	}
	return dbs, nil
}

func (s *DatabaseStore) CountByUserID(userID uint64) (int, error) {
	var count int
	err := s.db.Get(&count, "SELECT COUNT(*) FROM `databases` WHERE user_id = ?", userID)
	return count, err
}

func (s *DatabaseStore) Delete(id uint64) error {
	_, err := s.db.Exec("DELETE FROM `databases` WHERE id = ?", id)
	return err
}
