package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	agentclient "github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type PHPExtensionHandler struct {
	exts      *store.PHPExtensionStore
	users     *store.UserStore
	agentSock string
}

func NewPHPExtensionHandler(exts *store.PHPExtensionStore, users *store.UserStore, agentSock string) *PHPExtensionHandler {
	return &PHPExtensionHandler{exts: exts, users: users, agentSock: agentSock}
}

// AdminSeed inserts the default extension catalog if the table is empty.
// Idempotent — uses INSERT IGNORE so re-running is safe.
func (h *PHPExtensionHandler) AdminSeed(c *gin.Context) {
	existing, err := h.exts.List("")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(existing) > 0 {
		c.JSON(http.StatusOK, gin.H{"message": "already seeded", "count": len(existing)})
		return
	}

	type extDef struct{ name, ver string }
	defaults := []extDef{}
	for _, ver := range []string{"8.1", "8.2", "8.3", "8.4"} {
		for _, name := range []string{
			"bcmath", "curl", "gd", "intl", "mbstring", "mysqli",
			"opcache", "pdo_mysql", "redis", "soap", "xml", "zip",
			"imagick", "exif", "fileinfo", "iconv", "json",
		} {
			defaults = append(defaults, extDef{name, ver})
		}
	}

	count := 0
	for _, d := range defaults {
		ext := &store.PHPExtension{
			Name:       d.name,
			PHPVersion: d.ver,
			Enabled:    true,
		}
		if err := h.exts.Create(ext); err == nil {
			count++
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "seeded", "count": count})
}
// ?php_version=8.3. Admin-only.
func (h *PHPExtensionHandler) AdminList(c *gin.Context) {
	ver := c.Query("php_version")
	exts, err := h.exts.List(ver)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": exts})
}

// AdminUpdate toggles the global enabled flag for an extension.
// When disabled, no user can re-enable it (V20).
func (h *PHPExtensionHandler) AdminUpdate(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ext, err := h.exts.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "extension not found"})
		return
	}
	if err := h.exts.SetGlobal(id, req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Propagate global disable to running FPM pools (V44). Best-effort —
	// DB row is already updated; agent failures are non-fatal.
	if !req.Enabled {
		ac := agentclient.NewClient(h.agentSock)
		users, _ := h.exts.GetUsersWithExtEnabled(id)
		for _, u := range users {
			_ = ac.Call("phpfpm.disable_extension", map[string]interface{}{
				"username":    u.Username,
				"php_version": ext.PHPVersion,
				"ext_name":    ext.Name,
			}, nil)
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// AdminCreate adds a new extension to the global catalog.
func (h *PHPExtensionHandler) AdminCreate(c *gin.Context) {
	var req struct {
		Name       string `json:"name" binding:"required"`
		PHPVersion string `json:"php_version" binding:"required"`
		Enabled    bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ext := &store.PHPExtension{
		Name:       req.Name,
		PHPVersion: req.PHPVersion,
		Enabled:    req.Enabled,
	}
	if err := h.exts.Create(ext); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, ext)
}

// AdminDelete removes an extension from the global catalog.
func (h *PHPExtensionHandler) AdminDelete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := h.exts.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
// global catalog + per-user overrides. Requires ?php_version=X.Y.
func (h *PHPExtensionHandler) UserList(c *gin.Context) {
	userID := auth.GetUserID(c)
	ver := c.Query("php_version")
	if ver == "" {
		// Default to the user's configured shell PHP version.
		u, err := h.users.GetByID(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup user"})
			return
		}
		ver = u.PHPVersion
	}
	views, err := h.exts.GetUserState(userID, ver)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": views})
}

// UserUpdate toggles a per-user extension override. Rejects attempts to
// enable an admin-disabled extension (V20). Calls the agent to write the
// ini snippet and reload the pool (V21).
func (h *PHPExtensionHandler) UserUpdate(c *gin.Context) {
	userID := auth.GetUserID(c)
	var req struct {
		Name       string `json:"name"        binding:"required"`
		PHPVersion string `json:"php_version" binding:"required"`
		Enabled    bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Look up the global ext row so we have the ext_id and can enforce V20.
	exts, err := h.exts.List(req.PHPVersion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var extID uint64
	var adminEnabled bool
	for _, e := range exts {
		if e.Name == req.Name {
			extID = e.ID
			adminEnabled = e.Enabled
			break
		}
	}
	if extID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "extension not found for this PHP version"})
		return
	}
	// V20: user cannot enable an admin-disabled extension.
	if req.Enabled && !adminEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "extension is disabled by admin"})
		return
	}

	// Persist the override in DB.
	if err := h.exts.SetUserState(userID, extID, req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Look up username for the agent call.
	user, err := h.users.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup user"})
		return
	}

	// Call agent to write/remove the ini snippet and reload FPM (V21).
	ac := agentclient.NewClient(h.agentSock)
	rpc := "phpfpm.disable_extension"
	if req.Enabled {
		rpc = "phpfpm.enable_extension"
	}
	if err := ac.Call(rpc, map[string]interface{}{
		"username":    user.Username,
		"php_version": req.PHPVersion,
		"ext_name":    req.Name,
	}, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "agent: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}
