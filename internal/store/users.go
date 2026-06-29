package store

import (
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// jsonMarshal / jsonUnmarshal are local aliases so the recovery-code
// logic in ConsumeRecoveryCode doesn't need to import encoding/json
// at the call site — it's already imported here.
var jsonMarshal   = json.Marshal
var jsonUnmarshal = json.Unmarshal

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
	q := `INSERT INTO users (username, email, password_hash, role, linux_uid, package_id, status, terminal_enabled, backup_enabled, antivirus_enabled, php_version)
		  VALUES (:username, :email, :password_hash, :role, :linux_uid, :package_id, :status, :terminal_enabled, :backup_enabled, :antivirus_enabled, :php_version)`
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
	f.Sort = safeSort(f.Sort, "created_at", allowedUserSort)
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
	fields = filterAllowed(fields, allowedUserUpdate)
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

// UpdateLinuxUID is a typed setter used by the API handler when the agent
// provisions a different UID than originally proposed (e.g. /etc/passwd
// already had the requested UID). Kept separate from the dynamic Update
// path — linux_uid is intentionally excluded from the user-facing allowlist.
func (s *UserStore) UpdateLinuxUID(id uint64, uid int) error {
	_, err := s.db.Exec("UPDATE users SET linux_uid = ? WHERE id = ?", uid, id)
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

// SetTOTP persists the encrypted TOTP secret, enabled flag, and bcrypt-hashed
// recovery codes for a user. Pass empty strings + false to clear 2FA (V27).
func (s *UserStore) SetTOTP(id uint64, secretEnc string, enabled bool, recoveryCodes string) error {
	_, err := s.db.Exec(
		"UPDATE users SET totp_secret_enc = ?, totp_enabled = ?, totp_recovery_codes = ? WHERE id = ?",
		secretEnc, enabled, recoveryCodes, id)
	return err
}

// GetTOTPSecret returns the encrypted TOTP secret and enabled flag for a user.
func (s *UserStore) GetTOTPSecret(id uint64) (secretEnc string, enabled bool, err error) {
	var u User
	if err = s.db.Get(&u, "SELECT totp_secret_enc, totp_enabled FROM users WHERE id = ?", id); err != nil {
		return "", false, err
	}
	return u.TOTPSecretEnc.String, u.TOTPEnabled, nil
}

// ConsumeRecoveryCode checks whether code matches any stored recovery code hash
// for the user, removes it if found (single-use, V29), and returns true on match.
// Recovery codes are stored as a JSON array of bcrypt hashes.
func (s *UserStore) ConsumeRecoveryCode(id uint64, code string) (bool, error) {
	var u User
	if err := s.db.Get(&u, "SELECT totp_recovery_codes FROM users WHERE id = ?", id); err != nil {
		return false, err
	}
	if !u.TOTPRecoveryCodes.Valid || u.TOTPRecoveryCodes.String == "" {
		return false, nil
	}
	var hashes []string
	if err := jsonUnmarshal([]byte(u.TOTPRecoveryCodes.String), &hashes); err != nil {
		return false, fmt.Errorf("parse recovery codes: %w", err)
	}
	matchIdx := -1
	for i, h := range hashes {
		if bcrypt.CompareHashAndPassword([]byte(h), []byte(code)) == nil {
			matchIdx = i
			break
		}
	}
	if matchIdx < 0 {
		return false, nil
	}
	// Remove the consumed code and persist.
	hashes = append(hashes[:matchIdx], hashes[matchIdx+1:]...)
	updated, err := jsonMarshal(hashes)
	if err != nil {
		return false, fmt.Errorf("marshal recovery codes: %w", err)
	}
	_, err = s.db.Exec("UPDATE users SET totp_recovery_codes = ? WHERE id = ?", string(updated), id)
	return err == nil, err
}

// CountDomains returns the number of domain rows owned by the user. Used by
// GetUsage to populate the User Panel Dashboard cards.
func (s *UserStore) CountDomains(userID uint64) (int, error) {
	var n int
	err := s.db.Get(&n, "SELECT COUNT(*) FROM domains WHERE user_id = ?", userID)
	return n, err
}

// CountDatabases returns the number of database rows owned by the user. The
// `databases` table is backtick-quoted because it's a MySQL reserved word.
func (s *UserStore) CountDatabases(userID uint64) (int, error) {
	var n int
	err := s.db.Get(&n, "SELECT COUNT(*) FROM `databases` WHERE user_id = ?", userID)
	return n, err
}

func (s *UserStore) BumpTokenVersion(id uint64) error {
	_, err := s.db.Exec("UPDATE users SET token_version = token_version + 1 WHERE id = ?", id)
	return err
}

func (s *UserStore) GetTokenVersion(id uint64) (int, error) {
	var v int
	err := s.db.Get(&v, "SELECT token_version FROM users WHERE id = ?", id)
	return v, err
}
