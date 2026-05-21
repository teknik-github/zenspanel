package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type BackupHandler struct {
	backups       *store.BackupStore
	users         *store.UserStore
	databases     *store.DatabaseStore
	BackupTargets *store.BackupTargetStore
	Domains       *store.DomainStore
	homeBase      string
	backupBase    string
	agentSock     string
}

func NewBackupHandler(backups *store.BackupStore, users *store.UserStore, databases *store.DatabaseStore, homeBase, backupBase, agentSock string) *BackupHandler {
	return &BackupHandler{
		backups:    backups,
		users:      users,
		databases:  databases,
		homeBase:   homeBase,
		backupBase: backupBase,
		agentSock:  agentSock,
	}
}

func (h *BackupHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	row, err := h.backups.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}
	if auth.GetRole(c) == "user" && row.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (h *BackupHandler) List(c *gin.Context) {
	uid := auth.GetUserID(c)
	rows, err := h.backups.ListByUserID(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows})
}

func (h *BackupHandler) Create(c *gin.Context) {
	var req struct {
		Type string `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Type != "full" && req.Type != "db" && req.Type != "files" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be full, db, or files"})
		return
	}
	uid := auth.GetUserID(c)
	user, err := h.users.GetByID(uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if !user.BackupEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "backups not enabled for this user"})
		return
	}

	row := &store.Backup{
		UserID: uid,
		Type:   req.Type,
		Status: "pending",
	}
	if err := h.backups.Create(row); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Spawn the actual archival in a goroutine so the API returns
	// immediately. The frontend polls List() every 5s while a row is
	// pending or running, so progress lands in the UI without an open
	// connection.
	go h.runBackup(row.ID, uid, user.Username, req.Type)

	c.JSON(http.StatusAccepted, row)
}

func (h *BackupHandler) Download(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	row, err := h.backups.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}
	if auth.GetRole(c) == "user" && row.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if !row.FilePath.Valid || row.Status != "done" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "backup not ready"})
		return
	}
	c.FileAttachment(row.FilePath.String, filepath.Base(row.FilePath.String))
}

// Restore replays a backup over the user's current files and databases.
// This is destructive: existing files in the home directory are wiped
// before extract, and existing tables in each managed database are
// overwritten by the dumped state. Ownership is checked against the
// caller (admin override allowed). The actual work runs in a goroutine
// so the UI gets an immediate response and can poll the row's status
// the same way it polls Create.
func (h *BackupHandler) Restore(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	row, err := h.backups.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}
	if auth.GetRole(c) == "user" && row.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if row.Status != "done" || !row.FilePath.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "backup must be in 'done' status with a file path"})
		return
	}
	user, err := h.users.GetByID(row.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "owner user not found"})
		return
	}

	go h.runRestore(row.ID, row.UserID, user.Username, row.Type, row.FilePath.String)

	c.JSON(http.StatusAccepted, gin.H{
		"message":   "restore started",
		"backup_id": row.ID,
	})
}

// runRestore is the goroutine that drives a restore through to terminal
// state. Status transitions: done -> restoring -> done|restore_failed.
// We keep the row at "done" on success so the Restore button stays
// available for re-runs. Failures land on "restore_failed" with the
// underlying error in error_msg so the operator can see why before
// kicking it again.
func (h *BackupHandler) runRestore(backupID, userID uint64, username, kind, archivePath string) {
	_ = h.backups.UpdateStatus(backupID, "restoring", archivePath, 0, "")

	agentClient := agent.NewClient(h.agentSock)

	if kind == "files" || kind == "full" {
		if err := agentClient.Call("backup.restore_files", map[string]interface{}{
			"username":     username,
			"archive_path": archivePath,
		}, nil); err != nil {
			_ = h.backups.UpdateStatus(backupID, "restore_failed", archivePath, 0, "files: "+err.Error())
			return
		}
	}

	if kind == "db" || kind == "full" {
		dbs, err := h.databases.ListByUserID(userID)
		if err != nil {
			_ = h.backups.UpdateStatus(backupID, "restore_failed", archivePath, 0, "list dbs: "+err.Error())
			return
		}
		for _, db := range dbs {
			if err := agentClient.Call("backup.restore_db", map[string]interface{}{
				"db_name":      db.DBName,
				"archive_path": archivePath,
			}, nil); err != nil {
				_ = h.backups.UpdateStatus(backupID, "restore_failed", archivePath, 0, "db "+db.DBName+": "+err.Error())
				return
			}
		}
	}

	_ = h.backups.UpdateStatus(backupID, "done", archivePath, 0, "")
}

func (h *BackupHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	row, err := h.backups.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup not found"})
		return
	}
	if auth.GetRole(c) == "user" && row.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	if row.FilePath.Valid {
		if err := os.Remove(row.FilePath.String); err != nil && !os.IsNotExist(err) {
			log.Printf("backup delete: remove file %s: %v", row.FilePath.String, err)
		}
	}
	if err := h.backups.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// runBackup performs the actual archival in the background. Status
// transitions are persisted so the UI can show progress / error message.
//
// Backup files land at <backupBase>/<username>/<timestamp>-<kind>.tar.gz.
// "db"     — mysqldump of every panel-tracked DB the user owns
// "files"  — tar of the user's home directory
// "full"   — both, bundled together
//
// mysqldump is invoked as the panel system user; getting per-DB
// credentials would mean storing user passwords (we don't), so this
// relies on a `[mysqldump]` block in /etc/mysql/conf.d that authenticates
// as a backup-only MySQL user with read access to all schemas. That
// setup is documented in CONTRIBUTING.md.
func (h *BackupHandler) runBackup(id, userID uint64, username, kind string) {
	_ = h.backups.UpdateStatus(id, "running", "", 0, "")

	dir := filepath.Join(h.backupBase, username)
	if err := os.MkdirAll(dir, 0750); err != nil {
		_ = h.backups.UpdateStatus(id, "failed", "", 0, "mkdir: "+err.Error())
		return
	}
	stamp := time.Now().Format("20060102-150405")
	archivePath := filepath.Join(dir, fmt.Sprintf("%s-%s.tar.gz", stamp, kind))

	tmpDir, err := os.MkdirTemp("", "zp-backup-*")
	if err != nil {
		_ = h.backups.UpdateStatus(id, "failed", "", 0, "mktemp: "+err.Error())
		return
	}
	defer os.RemoveAll(tmpDir)

	tarArgs := []string{"-czf", archivePath}

	if kind == "db" || kind == "full" {
		dbs, err := h.databases.ListByUserID(userID)
		if err != nil {
			_ = h.backups.UpdateStatus(id, "failed", "", 0, "list dbs: "+err.Error())
			return
		}
		dumpPath := filepath.Join(tmpDir, "databases.sql")
		f, err := os.Create(dumpPath)
		if err != nil {
			_ = h.backups.UpdateStatus(id, "failed", "", 0, "create dump: "+err.Error())
			return
		}
		for _, db := range dbs {
			cmd := exec.Command("mysqldump", "--single-transaction", db.DBName)
			cmd.Stdout = f
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				f.Close()
				_ = h.backups.UpdateStatus(id, "failed", "", 0, "mysqldump "+db.DBName+": "+err.Error())
				return
			}
		}
		f.Close()
		tarArgs = append(tarArgs, "-C", tmpDir, "databases.sql")
	}

	if kind == "files" || kind == "full" {
		tarArgs = append(tarArgs, "-C", h.homeBase, username)
	}

	cmd := exec.Command("tar", tarArgs...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		_ = h.backups.UpdateStatus(id, "failed", "", 0, "tar: "+err.Error())
		return
	}

	info, err := os.Stat(archivePath)
	if err != nil {
		_ = h.backups.UpdateStatus(id, "failed", "", 0, "stat: "+err.Error())
		return
	}
	_ = h.backups.UpdateStatus(id, "done", archivePath, info.Size(), "")

	// Upload to all enabled remote targets (best-effort — local backup
	// is already marked done; remote failures are logged only).
	if h.BackupTargets != nil {
		targets, _ := h.BackupTargets.ListEnabled()
		ac := agent.NewClient(h.agentSock)
		for _, t := range targets {
			if err := ac.Call("backup.upload_s3", map[string]interface{}{
				"file_path":      archivePath,
				"target_id":      t.ID,
				"name":           t.Name,
				"type":           t.Type,
				"bucket":         t.Bucket,
				"prefix":         t.Prefix,
				"access_key":     t.AccessKey,
				"secret_key_enc": t.SecretKeyEnc,
				"region":         t.Region,
				"endpoint":       t.Endpoint,
			}, nil); err != nil {
				log.Printf("backup upload to target %q failed: %v", t.Name, err)
			}
		}
	}
}

// DomainBackup creates a backup of a single domain's docroot (V58).
// Scoped to docroot only — no full home, no cross-domain data.
func (h *BackupHandler) DomainBackup(c *gin.Context) {
	domainID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if h.Domains == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "domain store not configured"})
		return
	}
	domain, err := h.Domains.GetByID(domainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	if auth.GetRole(c) == "user" && domain.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	user, err := h.users.GetByID(domain.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup user"})
		return
	}

	// Create a backup row to track the job.
	row := &store.Backup{
		UserID: domain.UserID,
		Type:   "domain",
		Status: "pending",
	}
	if err := h.backups.Create(row); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	go h.runDomainBackup(row.ID, user.Username, domain.Domain, domain.DocumentRoot)
	c.JSON(http.StatusAccepted, gin.H{"job_id": row.ID, "backup_id": row.ID})
}

func (h *BackupHandler) runDomainBackup(id uint64, username, domainName, docRoot string) {
	_ = h.backups.UpdateStatus(id, "running", "", 0, "")

	var result struct {
		ArchivePath string `json:"archive_path"`
		Size        int64  `json:"size"`
	}
	if err := agent.NewClient(h.agentSock).Call("backup.domain", map[string]interface{}{
		"username":    username,
		"doc_root":    docRoot,
		"domain_name": domainName,
	}, &result); err != nil {
		_ = h.backups.UpdateStatus(id, "failed", "", 0, err.Error())
		return
	}
	_ = h.backups.UpdateStatus(id, "done", result.ArchivePath, result.Size, "")
}
