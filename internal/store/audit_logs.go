package store

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type AuditLogStore struct {
	db *sqlx.DB
}

func NewAuditLogStore(db *sqlx.DB) *AuditLogStore {
	return &AuditLogStore{db: db}
}

func (s *AuditLogStore) Create(l *AuditLog) error {
	q := `INSERT INTO audit_logs (user_id, action, resource, ip_address, user_agent, meta)
		  VALUES (:user_id, :action, :resource, :ip_address, :user_agent, :meta)`
	_, err := s.db.NamedExec(q, l)
	return err
}

type AuditLogFilter struct {
	UserID   *uint64
	Action   string
	DateFrom string
	DateTo   string
	Page     int
	Limit    int
}

func (s *AuditLogStore) List(f AuditLogFilter) ([]AuditLog, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 50
	}

	where := "WHERE 1=1"
	args := []interface{}{}

	if f.UserID != nil {
		where += " AND user_id = ?"
		args = append(args, *f.UserID)
	}
	if f.Action != "" {
		where += " AND action LIKE ?"
		args = append(args, "%"+f.Action+"%")
	}
	if f.DateFrom != "" {
		where += " AND created_at >= ?"
		args = append(args, f.DateFrom)
	}
	if f.DateTo != "" {
		where += " AND created_at <= ?"
		args = append(args, f.DateTo)
	}

	var total int
	if err := s.db.Get(&total, "SELECT COUNT(*) FROM audit_logs "+where, args...); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	offset := (f.Page - 1) * f.Limit
	query := fmt.Sprintf("SELECT * FROM audit_logs %s ORDER BY created_at DESC LIMIT ? OFFSET ?", where)
	args = append(args, f.Limit, offset)

	var logs []AuditLog
	if err := s.db.Select(&logs, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	return logs, total, nil
}
