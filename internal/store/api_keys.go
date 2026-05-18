package store

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type APIKeyStore struct {
	db *sqlx.DB
}

func NewAPIKeyStore(db *sqlx.DB) *APIKeyStore {
	return &APIKeyStore{db: db}
}

func (s *APIKeyStore) Create(k *APIKey) error {
	q := `INSERT INTO api_keys (name, key_hash, key_prefix, permissions, expires_at, created_by)
		  VALUES (:name, :key_hash, :key_prefix, :permissions, :expires_at, :created_by)`
	res, err := s.db.NamedExec(q, k)
	if err != nil {
		return fmt.Errorf("insert api key: %w", err)
	}
	id, _ := res.LastInsertId()
	k.ID = uint64(id)
	return nil
}

func (s *APIKeyStore) List() ([]APIKey, error) {
	var keys []APIKey
	if err := s.db.Select(&keys, "SELECT * FROM api_keys ORDER BY created_at DESC"); err != nil {
		return nil, fmt.Errorf("list api keys: %w", err)
	}
	return keys, nil
}

func (s *APIKeyStore) Delete(id uint64) error {
	_, err := s.db.Exec("DELETE FROM api_keys WHERE id = ?", id)
	return err
}

func (s *APIKeyStore) ValidateKey(rawKey string) (*APIKey, error) {
	if len(rawKey) < 16 {
		return nil, fmt.Errorf("invalid api key")
	}
	// Match what handlers/api_keys.go records on Create: the prefix is the
	// 8 hex chars immediately after "zp_live_", not the literal tag.
	prefix := rawKey[8:16]
	var keys []APIKey
	if err := s.db.Select(&keys, "SELECT * FROM api_keys WHERE key_prefix = ?", prefix); err != nil {
		return nil, fmt.Errorf("lookup api key: %w", err)
	}
	for _, k := range keys {
		if bcrypt.CompareHashAndPassword([]byte(k.KeyHash), []byte(rawKey)) == nil {
			if k.ExpiresAt.Valid && k.ExpiresAt.Time.Before(time.Now()) {
				return nil, fmt.Errorf("api key expired")
			}
			s.db.Exec("UPDATE api_keys SET last_used_at = NOW() WHERE id = ?", k.ID)
			return &k, nil
		}
	}
	return nil, fmt.Errorf("invalid api key")
}

func HashAPIKey(key string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	return string(b), err
}
