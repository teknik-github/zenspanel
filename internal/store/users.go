package store

import (
	"fmt"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type UserStore struct {
	db *sqlx.DB
}

func NewUserStore(db *sqlx.DB) *UserStore {
	return &UserStore{db: db}
}

type UserFilter struct {
	Search    string
	Status    string
	PackageID *uint64
	Page      int
	Limit     int
	Sort      string
	Order     string
}

func (s *UserStore) Create(u *User) error {
	q := `INSERT INTO users (username, email, password_hash, role, linux_uid, package_id, status, terminal_enabled, backup_enabled)
		  VALUES (:username, :email, :password_hash, :role, :linux_uid, :package_id, :status, :terminal_enabled, :backup_enabled)`
	res, err := s.db.NamedExec(q, u)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	id, _ := res.LastInsertId()
	u.ID = uint64(id)
	return nil
}

func (s *UserStore) GetByID(id uint64) (*User, error) {
	var u User
	if err := s.db.Get(&u, "SELECT * FROM users WHERE id = ?", id); err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

func (s *UserStore) GetByUsername(username string) (*User, error) {
	var u User
	if err := s.db.Get(&u, "SELECT * FROM users WHERE username = ?", username); err != nil {
		return nil, fmt.Errorf("get user by username: %w", err)
	}
	return &u, nil
}

func (s *UserStore) GetByEmail(email string) (*User, error) {
	var u User
	if err := s.db.Get(&u, "SELECT * FROM users WHERE email = ?", email); err != nil {
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return &u, nil
}

func (s *UserStore) List(f UserFilter) ([]User, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 {
		f.Limit = 20
	}
	if f.Sort == "" {
		f.Sort = "created_at"
	}
	if f.Order != "asc" {
		f.Order = "desc"
	}

	where := "WHERE 1=1"
	args := []interface{}{}

	if f.Search != "" {
		where += " AND (username LIKE ? OR email LIKE ?)"
		args = append(args, "%"+f.Search+"%", "%"+f.Search+"%")
	}
	if f.Status != "" {
		where += " AND status = ?"
		args = append(args, f.Status)
	}
	if f.PackageID != nil {
		where += " AND package_id = ?"
		args = append(args, *f.PackageID)
	}

	var total int
	if err := s.db.Get(&total, "SELECT COUNT(*) FROM users "+where, args...); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	offset := (f.Page - 1) * f.Limit
	query := fmt.Sprintf("SELECT * FROM users %s ORDER BY %s %s LIMIT ? OFFSET ?", where, f.Sort, f.Order)
	args = append(args, f.Limit, offset)

	var users []User
	if err := s.db.Select(&users, query, args...); err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	return users, total, nil
}

func (s *UserStore) Update(id uint64, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	q := "UPDATE users SET "
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

func (s *UserStore) Delete(id uint64) error {
	_, err := s.db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

func (s *UserStore) CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

func (s *UserStore) GetMaxLinuxUID() (int, error) {
	var maxUID int
	err := s.db.Get(&maxUID, "SELECT COALESCE(MAX(linux_uid), 9999) FROM users")
	return maxUID, err
}
