package store

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

type CronJob struct {
	ID         uint64       `db:"id"          json:"id"`
	UserID     uint64       `db:"user_id"     json:"user_id"`
	Expression string       `db:"expression"  json:"expression"`
	Command    string       `db:"command"     json:"command"`
	Enabled    bool         `db:"enabled"     json:"enabled"`
	LastRunAt  sql.NullTime `db:"last_run_at" json:"last_run_at"`
	CreatedAt  time.Time    `db:"created_at"  json:"created_at"`
	UpdatedAt  time.Time    `db:"updated_at"  json:"updated_at"`
}

type CronJobStore struct {
	db *sqlx.DB
}

func NewCronJobStore(db *sqlx.DB) *CronJobStore {
	return &CronJobStore{db: db}
}

func (s *CronJobStore) ListByUserID(userID uint64) ([]CronJob, error) {
	var jobs []CronJob
	err := s.db.Select(&jobs,
		"SELECT * FROM cron_jobs WHERE user_id = ? ORDER BY id", userID)
	if err != nil {
		return nil, fmt.Errorf("list cron_jobs: %w", err)
	}
	return jobs, nil
}

func (s *CronJobStore) GetByID(id uint64) (*CronJob, error) {
	var j CronJob
	if err := s.db.Get(&j, "SELECT * FROM cron_jobs WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("get cron_job: %w", err)
	}
	return &j, nil
}

func (s *CronJobStore) Create(j *CronJob) error {
	q := `INSERT INTO cron_jobs (user_id, expression, command, enabled)
	      VALUES (:user_id, :expression, :command, :enabled)`
	res, err := s.db.NamedExec(q, j)
	if err != nil {
		return fmt.Errorf("insert cron_job: %w", err)
	}
	id, _ := res.LastInsertId()
	j.ID = uint64(id)
	return nil
}

func (s *CronJobStore) Update(id uint64, fields map[string]interface{}) error {
	fields = filterAllowed(fields, allowedCronJobUpdate)
	if len(fields) == 0 {
		return nil
	}
	q := "UPDATE cron_jobs SET "
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

func (s *CronJobStore) Delete(id uint64) error {
	_, err := s.db.Exec("DELETE FROM cron_jobs WHERE id = ?", id)
	return err
}

// CountByUserID returns the number of cron jobs owned by the user.
// Used to enforce package.max_cron_jobs quota (V25).
func (s *CronJobStore) CountByUserID(userID uint64) (int, error) {
	var n int
	err := s.db.Get(&n, "SELECT COUNT(*) FROM cron_jobs WHERE user_id = ?", userID)
	return n, err
}
