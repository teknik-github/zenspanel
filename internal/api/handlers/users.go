package handlers

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type PackageHandler struct {
	packages *store.PackageStore
}

func NewPackageHandler(packages *store.PackageStore) *PackageHandler {
	return &PackageHandler{packages: packages}
}

// packageRequest accepts disk_quota and memory_limit in MB (V49).
// The UI sends MB; we convert to bytes before storing.
type packageRequest struct {
	Name               string `json:"name"`
	CPUQuota           int    `json:"cpu_quota"`
	DiskQuotaMB        int64  `json:"disk_quota_mb"`
	MemoryLimitMB      int64  `json:"memory_limit_mb"`
	MaxDomains         int    `json:"max_domains"`
	MaxDatabases       int    `json:"max_databases"`
	MaxCronJobs        int    `json:"max_cron_jobs"`
	MaxProcs           int    `json:"max_procs"`
	IOReadMbps         int64  `json:"io_read_mbps"`
	IOWriteMbps        int64  `json:"io_write_mbps"`
	AntivirusEnabled   bool   `json:"antivirus_enabled"`
	PHPVersionsAllowed string `json:"php_versions_allowed"`
	TerminalEnabled    bool   `json:"terminal_enabled"`
	BackupEnabled      bool   `json:"backup_enabled"`
}

func (r packageRequest) toPackage() store.Package {
	return store.Package{
		Name:               r.Name,
		CPUQuota:           r.CPUQuota,
		DiskQuota:          r.DiskQuotaMB * 1024 * 1024,
		MemoryLimit:        r.MemoryLimitMB * 1024 * 1024,
		MaxDomains:         r.MaxDomains,
		MaxDatabases:       r.MaxDatabases,
		MaxCronJobs:        r.MaxCronJobs,
		MaxProcs:           r.MaxProcs,
		IOReadBps:          r.IOReadMbps * 1024 * 1024,
		IOWriteBps:         r.IOWriteMbps * 1024 * 1024,
		AntivirusEnabled:   r.AntivirusEnabled,
		PHPVersionsAllowed: r.PHPVersionsAllowed,
		TerminalEnabled:    r.TerminalEnabled,
		BackupEnabled:      r.BackupEnabled,
	}
}

// packageAdminResponse includes all fields for admin use.
func packageAdminResponse(p store.Package) map[string]interface{} {
	return map[string]interface{}{
		"id":                   p.ID,
		"name":                 p.Name,
		"cpu_quota":            p.CPUQuota,
		"disk_quota":           p.DiskQuota,
		"disk_quota_mb":        p.DiskQuota / (1024 * 1024),
		"memory_limit":         p.MemoryLimit,
		"memory_limit_mb":      p.MemoryLimit / (1024 * 1024),
		"max_domains":          p.MaxDomains,
		"max_databases":        p.MaxDatabases,
		"max_cron_jobs":        p.MaxCronJobs,
		"max_procs":            p.MaxProcs,
		"io_read_bps":          p.IOReadBps,
		"io_read_mbps":         p.IOReadBps / (1024 * 1024),
		"io_write_bps":         p.IOWriteBps,
		"io_write_mbps":        p.IOWriteBps / (1024 * 1024),
		"antivirus_enabled":    p.AntivirusEnabled,
		"max_ftp_accounts":     p.MaxFTPAccounts,
		"php_versions_allowed": p.PHPVersionsAllowed,
		"terminal_enabled":     p.TerminalEnabled,
		"backup_enabled":       p.BackupEnabled,
		"created_at":           p.CreatedAt,
		"updated_at":           p.UpdatedAt,
	}
}

// packageUserResponse returns only customer-visible quota fields.
// Internal system limits (max_procs, io_*_bps, cpu_quota) are omitted.
func packageUserResponse(p store.Package) map[string]interface{} {
	return map[string]interface{}{
		"id":                   p.ID,
		"name":                 p.Name,
		"disk_quota_mb":        p.DiskQuota / (1024 * 1024),
		"memory_limit_mb":      p.MemoryLimit / (1024 * 1024),
		"max_domains":          p.MaxDomains,
		"max_databases":        p.MaxDatabases,
		"max_cron_jobs":        p.MaxCronJobs,
		"max_ftp_accounts":     p.MaxFTPAccounts,
		"antivirus_enabled":    p.AntivirusEnabled,
		"php_versions_allowed": p.PHPVersionsAllowed,
		"terminal_enabled":     p.TerminalEnabled,
		"backup_enabled":       p.BackupEnabled,
	}
}

// packageResponse is kept as an alias for admin use (backward compat).
func packageResponse(p store.Package) map[string]interface{} {
	return packageAdminResponse(p)
}

func (h *PackageHandler) List(c *gin.Context) {
	pkgs, err := h.packages.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	isAdmin := auth.GetRole(c) == "admin"
	resp := make([]map[string]interface{}, len(pkgs))
	for i, p := range pkgs {
		if isAdmin {
			resp[i] = packageAdminResponse(p)
		} else {
			resp[i] = packageUserResponse(p)
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": resp})
}

func (h *PackageHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	pkg, err := h.packages.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "package not found"})
		return
	}
	if auth.GetRole(c) == "admin" {
		c.JSON(http.StatusOK, packageAdminResponse(*pkg))
	} else {
		c.JSON(http.StatusOK, packageUserResponse(*pkg))
	}
}

func (h *PackageHandler) Create(c *gin.Context) {
	var req packageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pkg := req.toPackage()
	if err := h.packages.Create(&pkg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, packageResponse(pkg))
}

func (h *PackageHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req packageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pkg := req.toPackage()
	if err := h.packages.Update(id, &pkg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *PackageHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.packages.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// UserHandler
type UserHandler struct {
	users      *store.UserStore
	packages   *store.PackageStore
	domains    *store.DomainStore
	subdomains *store.SubdomainStore
	databases  *store.DatabaseStore
	ftp        *store.FTPAccountStore
	agentSock  string
}

func NewUserHandler(
	users *store.UserStore,
	packages *store.PackageStore,
	domains *store.DomainStore,
	subdomains *store.SubdomainStore,
	databases *store.DatabaseStore,
	ftp *store.FTPAccountStore,
	agentSock string,
) *UserHandler {
	return &UserHandler{
		users:      users,
		packages:   packages,
		domains:    domains,
		subdomains: subdomains,
		databases:  databases,
		ftp:        ftp,
		agentSock:  agentSock,
	}
}

func (h *UserHandler) List(c *gin.Context) {
	filter := store.UserFilter{
		Search: c.Query("search"),
		Status: c.Query("status"),
		Sort:   c.Query("sort"),
		Order:  c.Query("order"),
	}
	if p := c.Query("page"); p != "" {
		filter.Page, _ = strconv.Atoi(p)
	}
	if l := c.Query("limit"); l != "" {
		filter.Limit, _ = strconv.Atoi(l)
	}
	if pkgID := c.Query("package_id"); pkgID != "" {
		id, _ := strconv.ParseUint(pkgID, 10, 64)
		filter.PackageID = &id
	}

	users, total, err := h.users.List(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": users, "total": total})
}

func (h *UserHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	// non-admin can only get own profile
	if auth.GetRole(c) == "user" && auth.GetUserID(c) != id {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	user, err := h.users.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if auth.GetRole(c) == "admin" {
		c.JSON(http.StatusOK, userAdminResponse(user))
	} else {
		c.JSON(http.StatusOK, userSelfResponse(user))
	}
}

// userAdminResponse returns the full user object for admin callers.
func userAdminResponse(u *store.User) map[string]interface{} {
	return map[string]interface{}{
		"id":               u.ID,
		"username":         u.Username,
		"email":            u.Email,
		"role":             u.Role,
		"linux_uid":        u.LinuxUID,
		"package_id":       u.PackageID,
		"status":           u.Status,
		"terminal_enabled": u.TerminalEnabled,
		"backup_enabled":   u.BackupEnabled,
		"php_version":      u.PHPVersion,
		"totp_enabled":     u.TOTPEnabled,
		"created_at":       u.CreatedAt,
		"updated_at":       u.UpdatedAt,
	}
}

// userSelfResponse returns only the fields a user needs to see about
// their own account. Internal fields (linux_uid, role, status) are omitted
// — they are implementation details or admin-only state.
func userSelfResponse(u *store.User) map[string]interface{} {
	return map[string]interface{}{
		"id":               u.ID,
		"username":         u.Username,
		"email":            u.Email,
		"package_id":       u.PackageID,
		"terminal_enabled": u.TerminalEnabled,
		"backup_enabled":   u.BackupEnabled,
		"php_version":      u.PHPVersion,
		"totp_enabled":     u.TOTPEnabled,
		"created_at":       u.CreatedAt,
		"updated_at":       u.UpdatedAt,
	}
}

func (h *UserHandler) Create(c *gin.Context) {
	var req struct {
		Username        string `json:"username" binding:"required"`
		Email           string `json:"email" binding:"required"`
		Password        string `json:"password" binding:"required"`
		PackageID       uint64 `json:"package_id"`
		TerminalEnabled bool   `json:"terminal_enabled"`
		BackupEnabled   bool   `json:"backup_enabled"`
		PHPVersion      string `json:"php_version"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ValidateUsername(req.Username); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default to 8.3 — the version the installer auto-starts. Without
	// this, an admin who omits php_version would land on a PHP-FPM
	// socket that doesn't exist and every domain request would 502.
	if req.PHPVersion == "" {
		req.PHPVersion = "8.3"
	}

	hash, err := store.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash password failed"})
		return
	}

	maxUID, _ := h.users.GetMaxLinuxUID()
	newUID := maxUID + 1

	user := &store.User{
		Username:        req.Username,
		Email:           req.Email,
		PasswordHash:    hash,
		Role:            "user",
		LinuxUID:        newUID,
		Status:          "active",
		TerminalEnabled: req.TerminalEnabled,
		BackupEnabled:   req.BackupEnabled,
		PHPVersion:      req.PHPVersion,
	}
	if req.PackageID > 0 {
		user.PackageID.Int64 = int64(req.PackageID)
		user.PackageID.Valid = true
	}

	if err := h.users.Create(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Provision system resources via the agent. If the Linux user creation
	// fails the panel row is rolled back so the next attempt with the same
	// username is not blocked by the unique constraint. The agent picks a
	// free UID (it's the authoritative source — /etc/passwd may have UIDs
	// the DB doesn't know about), so we update the row to match what was
	// actually created.
	agentClient := agent.NewClient(h.agentSock)
	var createResp struct {
		UID int `json:"uid"`
	}
	if err := agentClient.Call("user.create", map[string]interface{}{
		"username":    user.Username,
		"uid":         user.LinuxUID,
		"php_version": user.PHPVersion,
	}, &createResp); err != nil {
		_ = h.users.Delete(user.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "provision linux user: " + err.Error()})
		return
	}
	if createResp.UID > 0 && createResp.UID != user.LinuxUID {
		_ = h.users.UpdateLinuxUID(user.ID, createResp.UID)
		user.LinuxUID = createResp.UID
	}

	// Provision a FileBrowser user record scoped to the panel user's
	// home. Without this, FileBrowser's proxy auth falls back to the
	// global admin account and the panel user sees every other user's
	// files. Failures are non-fatal — the user is still created and
	// can be added to FileBrowser manually later.
	provisionWarnings := []string{}
	if err := agentClient.Call("filebrowser.user_create", map[string]interface{}{
		"username": user.Username,
	}, nil); err != nil {
		provisionWarnings = append(provisionWarnings, "filebrowser: "+err.Error())
	}

	// Cgroup slice and PHP-FPM pool only make sense once a package picks the
	// limits and PHP version. Failures here are logged into the response but
	// do not roll back the row — the user can still log in, and an admin can
	// retry by reassigning the package.
	if user.PackageID.Valid {
		pkg, err := h.packages.GetByID(uint64(user.PackageID.Int64))
		if err == nil {
			if err := agentClient.Call("cgroups.create_slice", map[string]interface{}{
				"username":     user.Username,
				"cpu_quota":    pkg.CPUQuota,
				"memory_limit": pkg.MemoryLimit,
				"max_procs":    pkg.MaxProcs,
				"io_read_bps":  pkg.IOReadBps,
				"io_write_bps": pkg.IOWriteBps,
			}, nil); err != nil {
				provisionWarnings = append(provisionWarnings, "cgroups: "+err.Error())
			}
			if err := agentClient.Call("phpfpm.create_pool", map[string]interface{}{
				"username":    user.Username,
				"php_version": user.PHPVersion,
			}, nil); err != nil {
				provisionWarnings = append(provisionWarnings, "phpfpm: "+err.Error())
			}
			// Filesystem-level disk quota — kernel blocks writes past
			// the hard limit with EDQUOT. Soft-fail because the panel
			// is still usable without it; admin can re-trigger by
			// reassigning the package.
			if pkg.DiskQuota > 0 {
				if err := agentClient.Call("quota.set", map[string]interface{}{
					"username":   user.Username,
					"hard_bytes": pkg.DiskQuota,
				}, nil); err != nil {
					provisionWarnings = append(provisionWarnings, "quota: "+err.Error())
				}
				// Enforce DB quota — revoke INSERT/CREATE on user DBs if
				// total DB size already exceeds the package limit.
				_ = agentClient.Call("mysql.enforce_db_quota", map[string]interface{}{
					"db_user":    user.Username,
					"hard_bytes": pkg.DiskQuota,
				}, nil)
			}
		}
	}

	resp := gin.H{
		"id":               user.ID,
		"username":         user.Username,
		"email":            user.Email,
		"role":             user.Role,
		"linux_uid":        user.LinuxUID,
		"package_id":       user.PackageID,
		"status":           user.Status,
		"terminal_enabled": user.TerminalEnabled,
		"backup_enabled":   user.BackupEnabled,
		"php_version":      user.PHPVersion,
		"created_at":       user.CreatedAt,
	}
	if len(provisionWarnings) > 0 {
		resp["warnings"] = provisionWarnings
	}
	c.JSON(http.StatusCreated, resp)
}

func (h *UserHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var fields map[string]interface{}
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// remove protected fields
	delete(fields, "id")
	delete(fields, "linux_uid")
	delete(fields, "password_hash")

	// Capture php_version intent BEFORE the DB write, so we can re-seed
	// ~/bin/php after the row is updated. allowedUserUpdate gates the
	// DB write; the agent call is a side effect we trigger ourselves.
	newPHPVersion, _ := fields["php_version"].(string)

	if err := h.users.Update(id, fields); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if newPHPVersion != "" {
		user, uerr := h.users.GetByID(id)
		if uerr == nil {
			agentClient := agent.NewClient(h.agentSock)
			_ = agentClient.Call("user.setup_bin", map[string]interface{}{
				"username":    user.Username,
				"php_version": user.PHPVersion,
			}, nil)
		} else {
			// GetByID failed after a successful DB write — surface a warning
			// so the caller knows the shell symlink may be stale (B10).
			c.JSON(http.StatusOK, gin.H{
				"message": "updated",
				"warning": "php_version updated in DB but shell symlink could not be refreshed: " + uerr.Error(),
			})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	// Look up everything we need to tear down BEFORE deleting the row,
	// otherwise we lose the username + the FK joins that let us
	// enumerate the user's domains/databases.
	user, err := h.users.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	agentClient := agent.NewClient(h.agentSock)
	warnings := []string{}

	// 1. Tear down every domain the user owns. Each one removes the
	//    nginx vhost via the agent; we keep going on errors so a
	//    half-broken vhost doesn't block the whole delete. Subdomains
	//    are torn down before the parent so their .conf files come off
	//    nginx before the parent vhost goes away.
	domains, _ := h.domains.ListByUserID(id)
	subs, _ := h.subdomains.ListByUserID(id)
	for _, s := range subs {
		if err := agentClient.Call("nginx.delete_vhost", map[string]interface{}{
			"domain": s.FQDN,
		}, nil); err != nil {
			warnings = append(warnings, "nginx.delete_vhost subdomain "+s.FQDN+": "+err.Error())
		}
		if err := agentClient.Call("ssl.remove_cert", map[string]interface{}{
			"domain": s.FQDN,
		}, nil); err != nil {
			warnings = append(warnings, "ssl.remove_cert subdomain "+s.FQDN+": "+err.Error())
		}
	}
	for _, d := range domains {
		if err := agentClient.Call("nginx.delete_vhost", map[string]interface{}{
			"domain": d.Domain,
		}, nil); err != nil {
			warnings = append(warnings, "nginx.delete_vhost "+d.Domain+": "+err.Error())
		}
		if err := agentClient.Call("ssl.remove_cert", map[string]interface{}{
			"domain": d.Domain,
		}, nil); err != nil {
			warnings = append(warnings, "ssl.remove_cert "+d.Domain+": "+err.Error())
		}
	}

	// 2. Tear down every database the user owns. The agent drops both
	//    the schema and the MySQL user grant.
	dbs, _ := h.databases.ListByUserID(id)
	for _, db := range dbs {
		if err := agentClient.Call("mysql.drop_database", map[string]interface{}{
			"db_name": db.DBName,
			"db_user": db.DBUser,
		}, nil); err != nil {
			warnings = append(warnings, "mysql.drop_database "+db.DBName+": "+err.Error())
		}
	}

	// 3. PHP-FPM pool. We don't know which versions the user had pools
	//    for; try every supported version. DeletePool is idempotent and
	//    a no-op when the pool file doesn't exist.
	for _, ver := range []string{"8.3", "8.2", "8.1"} {
		_ = agentClient.Call("phpfpm.delete_pool", map[string]interface{}{
			"username":    user.Username,
			"php_version": ver,
		}, nil)
	}

	// 4. cgroup slice — RAM/CPU limits.
	if err := agentClient.Call("cgroups.delete_slice", map[string]interface{}{
		"username": user.Username,
	}, nil); err != nil {
		warnings = append(warnings, "cgroups.delete_slice: "+err.Error())
	}

	// 5. Filesystem-level quota.
	_ = agentClient.Call("quota.delete", map[string]interface{}{
		"username": user.Username,
	}, nil)

	// 6. FileBrowser record.
	if err := agentClient.Call("filebrowser.user_delete", map[string]interface{}{
		"username": user.Username,
	}, nil); err != nil {
		warnings = append(warnings, "filebrowser.user_delete: "+err.Error())
	}

	// 7. Linux user. This is the one that actually unblocks recreate
	//    with the same username — `useradd` refuses to create a user
	//    that's already in /etc/passwd. `userdel -r` also removes the
	//    home dir + mail spool. If this fails we DO surface it because
	//    leaving the Linux user behind is the bug the operator
	//    reported.
	if err := agentClient.Call("user.delete", map[string]interface{}{
		"username": user.Username,
	}, nil); err != nil {
		warnings = append(warnings, "user.delete: "+err.Error())
	}

	// 8. DB row last — by now every external system the row points
	//    into has been cleaned up, so a row delete failure is the only
	//    thing that would leave residue.
	if err := h.users.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete row: " + err.Error()})
		return
	}

	resp := gin.H{"message": "deleted"}
	if len(warnings) > 0 {
		resp["warnings"] = warnings
	}
	c.JSON(http.StatusOK, resp)
}

func (h *UserHandler) Suspend(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	user, err := h.users.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := h.users.Update(id, map[string]interface{}{"status": "suspended"}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ac := agent.NewClient(h.agentSock)
	warnings := []string{}

	// Suspend all nginx vhosts (V64)
	if err := ac.Call("nginx.suspend_all_vhosts", map[string]interface{}{
		"username": user.Username,
	}, nil); err != nil {
		warnings = append(warnings, "nginx: "+err.Error())
	}

	// Suspend all FTP accounts (V65)
	if h.ftp != nil {
		ftpAccounts, _ := h.ftp.ListByUserID(id)
		for _, a := range ftpAccounts {
			if err := ac.Call("ftp.suspend_user", map[string]interface{}{
				"ftp_username": a.FTPUsername,
			}, nil); err != nil {
				warnings = append(warnings, "ftp "+a.FTPUsername+": "+err.Error())
			}
		}
	}

	// Revoke all active sessions by bumping token_version (V63)
	if err := h.users.BumpTokenVersion(id); err != nil {
		warnings = append(warnings, "token_version: "+err.Error())
	}

	c.JSON(http.StatusOK, gin.H{"message": "suspended", "warnings": warnings})
}

func (h *UserHandler) Unsuspend(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	user, err := h.users.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if err := h.users.Update(id, map[string]interface{}{"status": "active"}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ac := agent.NewClient(h.agentSock)
	warnings := []string{}

	// Restore all nginx vhosts (V66)
	if err := ac.Call("nginx.unsuspend_all_vhosts", map[string]interface{}{
		"username": user.Username,
	}, nil); err != nil {
		warnings = append(warnings, "nginx: "+err.Error())
	}

	// Restore all FTP accounts (V66)
	if h.ftp != nil {
		ftpAccounts, _ := h.ftp.ListByUserID(id)
		for _, a := range ftpAccounts {
			if err := ac.Call("ftp.unsuspend_user", map[string]interface{}{
				"ftp_username": a.FTPUsername,
			}, nil); err != nil {
				warnings = append(warnings, "ftp "+a.FTPUsername+": "+err.Error())
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "unsuspended", "warnings": warnings})
}

func (h *UserHandler) ChangePackage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		PackageID uint64 `json:"package_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.users.Update(id, map[string]interface{}{"package_id": req.PackageID}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Push the new package's resource limits to the agent so cgroup +
	// disk quota match the package the user is now on. Failures are
	// non-fatal — the row was already updated, admin can retry.
	user, _ := h.users.GetByID(id)
	pkg, _ := h.packages.GetByID(req.PackageID)
	if user != nil && pkg != nil {
		agentClient := agent.NewClient(h.agentSock)
		_ = agentClient.Call("cgroups.update_slice", map[string]interface{}{
			"username":     user.Username,
			"cpu_quota":    pkg.CPUQuota,
			"memory_limit": pkg.MemoryLimit,
			"max_procs":    pkg.MaxProcs,
			"io_read_bps":  pkg.IOReadBps,
			"io_write_bps": pkg.IOWriteBps,
		}, nil)
		if pkg.DiskQuota > 0 {
			_ = agentClient.Call("quota.set", map[string]interface{}{
				"username":   user.Username,
				"hard_bytes": pkg.DiskQuota,
			}, nil)
			// Re-enforce DB quota with new package limit.
			_ = agentClient.Call("mysql.enforce_db_quota", map[string]interface{}{
				"db_user":    user.Username,
				"hard_bytes": pkg.DiskQuota,
			}, nil)
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "package updated"})
}

// AllMetrics returns cgroup metrics for every active user — used by the
// admin Resource Monitor to detect abuse. Fetches in parallel with a
// 3s timeout per user so a stuck cgroup doesn't block the whole response.
func (h *UserHandler) AllMetrics(c *gin.Context) {
	users, _, err := h.users.List(store.UserFilter{
		Status: "active",
		Limit:  200,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type userMetric struct {
		ID       uint64  `json:"id"`
		Username string  `json:"username"`
		RAMUsed  int64   `json:"ram_used"`
		RAMMax   int64   `json:"ram_max"`
		DiskUsed int64   `json:"disk_used"`
		DiskMax  int64   `json:"disk_max"`
		CPUPct   float64 `json:"cpu_pct"`
	}

	results := make([]userMetric, len(users))
	var wg sync.WaitGroup
	for i, u := range users {
		wg.Add(1)
		go func(idx int, user store.User) {
			defer wg.Done()
			m := userMetric{
				ID:       user.ID,
				Username: user.Username,
			}
			// Get package limits.
			if user.PackageID.Valid {
				if pkg, err := h.packages.GetByID(uint64(user.PackageID.Int64)); err == nil {
					m.RAMMax = pkg.MemoryLimit
					m.DiskMax = pkg.DiskQuota
				}
			}
			// Get live cgroup metrics.
			var metrics struct {
				RAMUsed  int64   `json:"ram_used"`
				DiskUsed int64   `json:"disk_used"`
				CPUPct   float64 `json:"cpu_pct"`
			}
			_ = agent.NewClient(h.agentSock).Call("cgroups.read_metrics", map[string]interface{}{
				"username": user.Username,
			}, &metrics)
			m.RAMUsed = metrics.RAMUsed
			m.DiskUsed = metrics.DiskUsed
			m.CPUPct = metrics.CPUPct
			results[idx] = m
		}(i, u)
	}
	wg.Wait()

	c.JSON(http.StatusOK, gin.H{"data": results})
}
// the User Panel Dashboard expects. RAM and disk are read from the agent
// (cgroup v2 + du); failures fall through to 0 so the dashboard always
// renders even when a user hasn't been provisioned yet or the agent is
// briefly unreachable.
func (h *UserHandler) GetUsage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	// Ownership check — without this, any logged-in user can iterate
	// :id and read every other user's quota + live cgroup metrics.
	// Mirrors the pattern in users.Get.
	if auth.GetRole(c) == "user" && id != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	user, err := h.users.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	domainsUsed, _ := h.users.CountDomains(id)
	databasesUsed, _ := h.users.CountDatabases(id)

	maxDomains, maxDatabases := 0, 0
	maxDisk, maxRAM := int64(0), int64(0)
	if user.PackageID.Valid {
		if pkg, err := h.packages.GetByID(uint64(user.PackageID.Int64)); err == nil {
			maxDomains = pkg.MaxDomains
			maxDatabases = pkg.MaxDatabases
			maxDisk = pkg.DiskQuota
			maxRAM = pkg.MemoryLimit
		}
	}

	var metrics struct {
		RAMUsed  int64   `json:"ram_used"`
		DiskUsed int64   `json:"disk_used"`
		CPUPct   float64 `json:"cpu_pct"`
	}
	agentClient := agent.NewClient(h.agentSock)
	_ = agentClient.Call("cgroups.read_metrics", map[string]interface{}{
		"username": user.Username,
	}, &metrics)

	// Add MySQL DB size to disk usage so the dashboard reflects total
	// storage (files + databases) against the package quota.
	var dbSizeRes struct {
		SizeBytes int64 `json:"size_bytes"`
	}
	if len(user.Username) > 0 {
		_ = agentClient.Call("mysql.get_db_size", map[string]interface{}{
			"db_user": user.Username,
		}, &dbSizeRes)
	}
	totalDiskUsed := metrics.DiskUsed + dbSizeRes.SizeBytes

	c.JSON(http.StatusOK, gin.H{
		"user_id": id,
		"usage": gin.H{
			"domains":   gin.H{"used": domainsUsed, "max": maxDomains},
			"databases": gin.H{"used": databasesUsed, "max": maxDatabases},
			"disk":      gin.H{"used": totalDiskUsed, "max": maxDisk, "files": metrics.DiskUsed, "db": dbSizeRes.SizeBytes},
			"ram":       gin.H{"used": metrics.RAMUsed, "max": maxRAM},
			"cpu":       gin.H{"used": metrics.CPUPct, "max": 100},
		},
	})
}
