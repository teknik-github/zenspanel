package store

import (
	"database/sql"
	"time"
)

type User struct {
	ID              uint64         `db:"id"`
	Username        string         `db:"username"`
	Email           string         `db:"email"`
	PasswordHash    string         `db:"password_hash"`
	Role            string         `db:"role"`
	LinuxUID        int            `db:"linux_uid"`
	PackageID       sql.NullInt64  `db:"package_id"`
	Status          string         `db:"status"`
	TerminalEnabled bool           `db:"terminal_enabled"`
	BackupEnabled   bool           `db:"backup_enabled"`
	CreatedAt       time.Time      `db:"created_at"`
	UpdatedAt       time.Time      `db:"updated_at"`
}

type Package struct {
	ID                  uint64    `db:"id"`
	Name                string    `db:"name"`
	CPUQuota            int       `db:"cpu_quota"`
	MemoryLimit         int64     `db:"memory_limit"`
	DiskQuota           int64     `db:"disk_quota"`
	MaxDomains          int       `db:"max_domains"`
	MaxDatabases        int       `db:"max_databases"`
	PHPVersionsAllowed  string    `db:"php_versions_allowed"`
	TerminalEnabled     bool      `db:"terminal_enabled"`
	BackupEnabled       bool      `db:"backup_enabled"`
	CreatedAt           time.Time `db:"created_at"`
	UpdatedAt           time.Time `db:"updated_at"`
}

type Domain struct {
	ID           uint64         `db:"id"`
	UserID       uint64         `db:"user_id"`
	Domain       string         `db:"domain"`
	DocumentRoot string         `db:"document_root"`
	PHPVersion   string         `db:"php_version"`
	SSLType      string         `db:"ssl_type"`
	SSLExpiresAt sql.NullTime   `db:"ssl_expires_at"`
	Status       string         `db:"status"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

type Database struct {
	ID        uint64    `db:"id"`
	UserID    uint64    `db:"user_id"`
	DBName    string    `db:"db_name"`
	DBUser    string    `db:"db_user"`
	CreatedAt time.Time `db:"created_at"`
}

type ResourceLimit struct {
	ID           uint64    `db:"id"`
	UserID       uint64    `db:"user_id"`
	CPUQuota     int       `db:"cpu_quota"`
	MemoryLimit  int64     `db:"memory_limit"`
	DiskQuota    int64     `db:"disk_quota"`
	MaxDomains   int       `db:"max_domains"`
	MaxDatabases int       `db:"max_databases"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type PHPVersion struct {
	ID        uint64    `db:"id"`
	Version   string    `db:"version"`
	FPMSocket string    `db:"fpm_socket"`
	Enabled   bool      `db:"enabled"`
	CreatedAt time.Time `db:"created_at"`
}

type SSLCertificate struct {
	ID        uint64    `db:"id"`
	DomainID  uint64    `db:"domain_id"`
	Type      string    `db:"type"`
	CertPath  string    `db:"cert_path"`
	KeyPath   string    `db:"key_path"`
	ExpiresAt time.Time `db:"expires_at"`
	AutoRenew bool      `db:"auto_renew"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Backup struct {
	ID        uint64         `db:"id"`
	UserID    uint64         `db:"user_id"`
	Type      string         `db:"type"`
	Status    string         `db:"status"`
	FilePath  sql.NullString `db:"file_path"`
	SizeBytes sql.NullInt64  `db:"size_bytes"`
	ErrorMsg  sql.NullString `db:"error_msg"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

type APIKey struct {
	ID          uint64         `db:"id"`
	Name        string         `db:"name"`
	KeyHash     string         `db:"key_hash"`
	KeyPrefix   string         `db:"key_prefix"`
	Permissions string         `db:"permissions"`
	LastUsedAt  sql.NullTime   `db:"last_used_at"`
	ExpiresAt   sql.NullTime   `db:"expires_at"`
	CreatedBy   uint64         `db:"created_by"`
	CreatedAt   time.Time      `db:"created_at"`
}

type AuditLog struct {
	ID        uint64         `db:"id"`
	UserID    sql.NullInt64  `db:"user_id"`
	Action    string         `db:"action"`
	Resource  sql.NullString `db:"resource"`
	IPAddress string         `db:"ip_address"`
	UserAgent sql.NullString `db:"user_agent"`
	Meta      sql.NullString `db:"meta"`
	CreatedAt time.Time      `db:"created_at"`
}
