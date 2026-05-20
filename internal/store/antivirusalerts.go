package store

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type AntivirusAlert struct {
	ID         uint64    `db:"id"          json:"id"`
	UserID     uint64    `db:"user_id"     json:"user_id"`
	Path       string    `db:"path"        json:"path"`
	Threat     string    `db:"threat"      json:"threat"`
	DetectedAt time.Time `db:"detected_at" json:"detected_at"`
}

type AntivirusAlertStore struct {
	db *sqlx.DB
}

func NewAntivirusAlertStore(db *sqlx.DB) *AntivirusAlertStore {
	return &AntivirusAlertStore{db: db}
}

func (s *AntivirusAlertStore) Create(a *AntivirusAlert) error {
	q := `INSERT INTO antivirus_alerts (user_id, path, threat, detected_at)
	      VALUES (:user_id, :path, :threat, :detected_at)`
	res, err := s.db.NamedExec(q, a)
	if err != nil {
		return fmt.Errorf("insert antivirus_alert: %w", err)
	}
	id, _ := res.LastInsertId()
	a.ID = uint64(id)
	return nil
}

func (s *AntivirusAlertStore) ListByUserID(userID uint64, limit int) ([]AntivirusAlert, error) {
	if limit <= 0 {
		limit = 50
	}
	var alerts []AntivirusAlert
	err := s.db.Select(&alerts,
		"SELECT * FROM antivirus_alerts WHERE user_id = ? ORDER BY detected_at DESC LIMIT ?",
		userID, limit)
	if err != nil {
		return nil, fmt.Errorf("list antivirus_alerts: %w", err)
	}
	return alerts, nil
}

func (s *AntivirusAlertStore) DeleteByUserID(userID uint64) error {
	_, err := s.db.Exec("DELETE FROM antivirus_alerts WHERE user_id = ?", userID)
	return err
}
