package store

import (
	"database/sql"
	"time"
)

type User struct {
	ID              uint64    `db:"id" json:"id"`
	Username        string    `db:"username" json:"username"`
	Email           string    `db:"email" json:"email"`
	PasswordHash    string    `db:"password_hash" json:"-"`
	Role            string    `db:"role" json:"role"`
	LinuxUID        int       `db:"linux_uid" json:"linux_uid"`
	PackageID       NullInt64 `db:"package_id" json:"package_id"`
	Status          string    `db:"status" json:"status"`
	TerminalEnabled bool      `db:"terminal_enabled" json:"terminal_enabled"`
	BackupEnabled   bool      `db:"backup_enabled" json:"backup_enabled"`
	PHPVersion      string    `db:"php_version" json:"php_version"`
	CreatedAt       time.Time `db:"created_at" json:"created_at"`
	UpdatedAt       time.Time `db:"updated_at" json:"updated_at"`
}

type Package struct {
	ID                 uint64    `db:"id" json:"id"`
	Name               string    `db:"name" json:"name"`
	CPUQuota           int       `db:"cpu_quota" json:"cpu_quota"`
	MemoryLimit        int64     `db:"memory_limit" json:"memory_limit"`
	DiskQuota          int64     `db:"disk_quota" json:"disk_quota"`
	MaxDomains         int       `db:"max_domains" json:"max_domains"`
	MaxDatabases       int       `db:"max_databases" json:"max_databases"`
	MaxCronJobs        int       `db:"max_cron_jobs" json:"max_cron_jobs"`
	PHPVersionsAllowed string    `db:"php_versions_allowed" json:"php_versions_allowed"`
	TerminalEnabled    bool      `db:"terminal_enabled" json:"terminal_enabled"`
	BackupEnabled      bool      `db:"backup_enabled" json:"backup_enabled"`
	CreatedAt          time.Time `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time `db:"updated_at" json:"updated_at"`
}

type Domain struct {
	ID           uint64       `db:"id" json:"id"`
	UserID       uint64       `db:"user_id" json:"user_id"`
	Domain       string       `db:"domain" json:"domain"`
	DocumentRoot string       `db:"document_root" json:"document_root"`
	PHPVersion   string       `db:"php_version" json:"php_version"`
	SSLType      string       `db:"ssl_type" json:"ssl_type"`
	SSLExpiresAt sql.NullTime `db:"ssl_expires_at" json:"ssl_expires_at"`
	Status       string       `db:"status" json:"status"`
	CreatedAt    time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time    `db:"updated_at" json:"updated_at"`
}

// Subdomain is a child of Domain. fqdn = subdomain + "." + parent.Domain
// is materialised at create time so listing/lookup don't have to JOIN.
// Same shape as Domain otherwise — own nginx vhost, own document root,
// own SSL state — but borrows the parent's PHP-FPM pool (pools are
// per-user, not per-vhost).
type Subdomain struct {
	ID             uint64       `db:"id" json:"id"`
	UserID         uint64       `db:"user_id" json:"user_id"`
	ParentDomainID uint64       `db:"parent_domain_id" json:"parent_domain_id"`
	Subdomain      string       `db:"subdomain" json:"subdomain"`
	FQDN           string       `db:"fqdn" json:"fqdn"`
	DocumentRoot   string       `db:"document_root" json:"document_root"`
	PHPVersion     string       `db:"php_version" json:"php_version"`
	SSLType        string       `db:"ssl_type" json:"ssl_type"`
	SSLExpiresAt   sql.NullTime `db:"ssl_expires_at" json:"ssl_expires_at"`
	Status         string       `db:"status" json:"status"`
	CreatedAt      time.Time    `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time    `db:"updated_at" json:"updated_at"`
}

type Database struct {
	ID        uint64    `db:"id" json:"id"`
	UserID    uint64    `db:"user_id" json:"user_id"`
	DBName    string    `db:"db_name" json:"db_name"`
	DBUser    string    `db:"db_user" json:"db_user"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type ResourceLimit struct {
	ID           uint64    `db:"id" json:"id"`
	UserID       uint64    `db:"user_id" json:"user_id"`
	CPUQuota     int       `db:"cpu_quota" json:"cpu_quota"`
	MemoryLimit  int64     `db:"memory_limit" json:"memory_limit"`
	DiskQuota    int64     `db:"disk_quota" json:"disk_quota"`
	MaxDomains   int       `db:"max_domains" json:"max_domains"`
	MaxDatabases int       `db:"max_databases" json:"max_databases"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type PHPVersion struct {
	ID        uint64    `db:"id" json:"id"`
	Version   string    `db:"version" json:"version"`
	FPMSocket string    `db:"fpm_socket" json:"fpm_socket"`
	Enabled   bool      `db:"enabled" json:"enabled"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type SSLCertificate struct {
	ID        uint64    `db:"id" json:"id"`
	DomainID  uint64    `db:"domain_id" json:"domain_id"`
	Type      string    `db:"type" json:"type"`
	CertPath  string    `db:"cert_path" json:"cert_path"`
	KeyPath   string    `db:"key_path" json:"key_path"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
	AutoRenew bool      `db:"auto_renew" json:"auto_renew"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type Backup struct {
	ID        uint64         `db:"id" json:"id"`
	UserID    uint64         `db:"user_id" json:"user_id"`
	Type      string         `db:"type" json:"type"`
	Status    string         `db:"status" json:"status"`
	FilePath  sql.NullString `db:"file_path" json:"file_path"`
	SizeBytes sql.NullInt64  `db:"size_bytes" json:"size_bytes"`
	ErrorMsg  sql.NullString `db:"error_msg" json:"error_msg"`
	CreatedAt time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt time.Time      `db:"updated_at" json:"updated_at"`
}

type APIKey struct {
	ID          uint64       `db:"id" json:"id"`
	Name        string       `db:"name" json:"name"`
	KeyHash     string       `db:"key_hash" json:"-"`
	KeyPrefix   string       `db:"key_prefix" json:"key_prefix"`
	Permissions string       `db:"permissions" json:"permissions"`
	LastUsedAt  sql.NullTime `db:"last_used_at" json:"last_used_at"`
	ExpiresAt   sql.NullTime `db:"expires_at" json:"expires_at"`
	CreatedBy   uint64       `db:"created_by" json:"created_by"`
	CreatedAt   time.Time    `db:"created_at" json:"created_at"`
}

type AuditLog struct {
	ID        uint64         `db:"id" json:"id"`
	UserID    sql.NullInt64  `db:"user_id" json:"user_id"`
	Action    string         `db:"action" json:"action"`
	Resource  sql.NullString `db:"resource" json:"resource"`
	IPAddress string         `db:"ip_address" json:"ip_address"`
	UserAgent sql.NullString `db:"user_agent" json:"user_agent"`
	Meta      sql.NullString `db:"meta" json:"meta"`
	CreatedAt time.Time      `db:"created_at" json:"created_at"`
}
