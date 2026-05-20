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
	domains   *store.DomainStore
	users     *store.UserStore
	agentSock string
}

func NewInstallerHandler(domains *store.DomainStore, users *store.UserStore, agentSock string) *InstallerHandler {
	return &InstallerHandler{domains: domains, users: users, agentSock: agentSock}
}

// ListApps returns the static app catalog.
func (h *InstallerHandler) ListApps(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": agentinstaller.Catalog})
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

	domain, err := h.domains.GetByID(req.DomainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	// Ownership check — users can only install into their own domains.
	if auth.GetRole(c) == "user" && domain.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	user, err := h.users.GetByID(domain.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup user"})
		return
	}

	// Generate a unique job ID.
	b := make([]byte, 8)
	rand.Read(b)
	jobID := hex.EncodeToString(b)

	dbHost := "127.0.0.1"
	siteURL := "http://" + domain.Domain

	var result map[string]interface{}
	if err := agentclient.NewClient(h.agentSock).Call("installer.run", map[string]interface{}{
		"job_id":    jobID,
		"app_id":    req.AppID,
		"username":  user.Username,
		"doc_root":  domain.DocumentRoot,
		"db_name":   req.DBName,
		"db_user":   req.DBUser,
		"db_pass":   req.DBPass,
		"db_host":   dbHost,
		"site_url":  siteURL,
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
