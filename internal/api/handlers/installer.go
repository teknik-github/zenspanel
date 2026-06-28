package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
	agentinstaller "github.com/zenspanel/zenspanel/agent/installer"
	agentclient "github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type InstallerHandler struct {
	domains       *store.DomainStore
	users         *store.UserStore
	installerApps *store.InstallerAppStore
	agentSock     string
}

func NewInstallerHandler(domains *store.DomainStore, users *store.UserStore, installerApps *store.InstallerAppStore, agentSock string) *InstallerHandler {
	return &InstallerHandler{domains: domains, users: users, installerApps: installerApps, agentSock: agentSock}
}

// ListApps returns the subset of the catalog that the admin has enabled.
func (h *InstallerHandler) ListApps(c *gin.Context) {
	enabled, err := h.installerApps.EnabledMap()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var visible []agentinstaller.App
	for _, app := range agentinstaller.Catalog {
		if enabled[app.ID] {
			visible = append(visible, app)
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": visible})
}

// AdminListApps returns the full catalog merged with each app's enabled state.
func (h *InstallerHandler) AdminListApps(c *gin.Context) {
	enabled, err := h.installerApps.EnabledMap()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type appWithEnabled struct {
		agentinstaller.App
		Enabled bool `json:"enabled"`
	}
	out := make([]appWithEnabled, 0, len(agentinstaller.Catalog))
	for _, app := range agentinstaller.Catalog {
		out = append(out, appWithEnabled{App: app, Enabled: enabled[app.ID]})
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

// AdminSetEnabled toggles the global enabled flag for one installer app.
func (h *InstallerHandler) AdminSetEnabled(c *gin.Context) {
	slug := c.Param("slug")
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.installerApps.SetEnabled(slug, req.Enabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"slug": slug, "enabled": req.Enabled})
}

// Install starts an async installation job. Returns a job_id to poll.
func (h *InstallerHandler) Install(c *gin.Context) {
	userID := auth.GetUserID(c)
	var req struct {
		AppID     string `json:"app_id"    binding:"required"`
		DomainID  uint64 `json:"domain_id" binding:"required"`
		DBName    string `json:"db_name"`
		DBUser    string `json:"db_user"`
		DBPass    string `json:"db_pass"`
		Overwrite bool   `json:"overwrite"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify the requested app is globally enabled.
	enabled, err := h.installerApps.EnabledMap()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !enabled[req.AppID] {
		c.JSON(http.StatusForbidden, gin.H{"error": "installer not available"})
		return
	}

	domain, err := h.domains.GetByID(req.DomainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	if auth.GetRole(c) == "user" && domain.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	user, err := h.users.GetByID(domain.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup user"})
		return
	}

	b := make([]byte, 8)
	rand.Read(b) //nolint:errcheck
	jobID := hex.EncodeToString(b)

	var result map[string]interface{}
	if err := agentclient.NewClient(h.agentSock).Call("installer.run", map[string]interface{}{
		"job_id":    jobID,
		"app_id":    req.AppID,
		"username":  user.Username,
		"doc_root":  domain.DocumentRoot,
		"db_name":   req.DBName,
		"db_user":   req.DBUser,
		"db_pass":   req.DBPass,
		"db_host":   "127.0.0.1",
		"site_url":  "http://" + domain.Domain,
		"overwrite": req.Overwrite,
	}, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"job_id": jobID})
}

// Status returns the current status of an install job.
func (h *InstallerHandler) Status(c *gin.Context) {
	jobID := c.Param("job_id")
	var status map[string]interface{}
	if err := agentclient.NewClient(h.agentSock).Call("installer.status", map[string]interface{}{
		"job_id": jobID,
	}, &status); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}
