package handlers

import (
	"net/http"
	"strconv"

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

func (h *PackageHandler) List(c *gin.Context) {
	pkgs, err := h.packages.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": pkgs})
}

func (h *PackageHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	pkg, err := h.packages.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "package not found"})
		return
	}
	c.JSON(http.StatusOK, pkg)
}

func (h *PackageHandler) Create(c *gin.Context) {
	var pkg store.Package
	if err := c.ShouldBindJSON(&pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.packages.Create(&pkg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, pkg)
}

func (h *PackageHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var pkg store.Package
	if err := c.ShouldBindJSON(&pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
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
	users     *store.UserStore
	packages  *store.PackageStore
	domains   *store.DomainStore
	databases *store.DatabaseStore
	agentSock string
}

func NewUserHandler(
	users *store.UserStore,
	packages *store.PackageStore,
	domains *store.DomainStore,
	databases *store.DatabaseStore,
	agentSock string,
) *UserHandler {
	return &UserHandler{
		users:     users,
		packages:  packages,
		domains:   domains,
		databases: databases,
		agentSock: agentSock,
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
	if auth.GetRole(c) != "admin" && auth.GetUserID(c) != id {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	user, err := h.users.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) Create(c *gin.Context) {
	var req struct {
		Username        string `json:"username" binding:"required"`
		Email           string `json:"email" binding:"required"`
		Password        string `json:"password" binding:"required"`
		PackageID       uint64 `json:"package_id"`
		TerminalEnabled bool   `json:"terminal_enabled"`
		BackupEnabled   bool   `json:"backup_enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ValidateUsername(req.Username); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
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
		"username": user.Username,
		"uid":      user.LinuxUID,
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
			}, nil); err != nil {
				provisionWarnings = append(provisionWarnings, "cgroups: "+err.Error())
			}
			if err := agentClient.Call("phpfpm.create_pool", map[string]interface{}{
				"username":    user.Username,
				"php_version": "8.3",
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
	if err := h.users.Update(id, fields); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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
	//    half-broken vhost doesn't block the whole delete.
	domains, _ := h.domains.ListByUserID(id)
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
	if err := h.users.Update(id, map[string]interface{}{"status": "suspended"}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "suspended"})
}

func (h *UserHandler) Unsuspend(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.users.Update(id, map[string]interface{}{"status": "active"}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "unsuspended"})
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
		}, nil)
		if pkg.DiskQuota > 0 {
			_ = agentClient.Call("quota.set", map[string]interface{}{
				"username":   user.Username,
				"hard_bytes": pkg.DiskQuota,
			}, nil)
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "package updated"})
}

// GetUsage returns the resource usage for a user in the {used, max} shape
// the User Panel Dashboard expects. RAM and disk are read from the agent
// (cgroup v2 + du); failures fall through to 0 so the dashboard always
// renders even when a user hasn't been provisioned yet or the agent is
// briefly unreachable.
func (h *UserHandler) GetUsage(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

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

	c.JSON(http.StatusOK, gin.H{
		"user_id": id,
		"usage": gin.H{
			"domains":   gin.H{"used": domainsUsed, "max": maxDomains},
			"databases": gin.H{"used": databasesUsed, "max": maxDatabases},
			"disk":      gin.H{"used": metrics.DiskUsed, "max": maxDisk},
			"ram":       gin.H{"used": metrics.RAMUsed, "max": maxRAM},
			"cpu":       gin.H{"used": metrics.CPUPct, "max": 100},
		},
	})
}
