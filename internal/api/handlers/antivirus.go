package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
	agentclient "github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type AntivirusHandler struct {
	users     *store.UserStore
	agentSock string
}

func NewAntivirusHandler(users *store.UserStore, agentSock string) *AntivirusHandler {
	return &AntivirusHandler{users: users, agentSock: agentSock}
}

// DaemonStatus returns whether the ClamAV daemon is running.
func (h *AntivirusHandler) DaemonStatus(c *gin.Context) {
	var result map[string]interface{}
	if err := agentclient.NewClient(h.agentSock).Call("antivirus.status", map[string]interface{}{
		"job_id": "",
	}, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// Scan starts an async antivirus scan of the calling user's home directory.
// Path is relative to user home; empty = full home scan. Agent enforces
// the path jail (V40) — scan cannot escape user home.
func (h *AntivirusHandler) Scan(c *gin.Context) {
	userID := auth.GetUserID(c)
	user, err := h.users.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup user"})
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	_ = c.ShouldBindJSON(&req)

	b := make([]byte, 8)
	rand.Read(b)
	jobID := hex.EncodeToString(b)

	var result map[string]interface{}
	if err := agentclient.NewClient(h.agentSock).Call("antivirus.scan", map[string]interface{}{
		"job_id":   jobID,
		"username": user.Username,
		"path":     req.Path,
	}, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"job_id": jobID})
}

// ScanStatus returns the current status of a scan job.
func (h *AntivirusHandler) ScanStatus(c *gin.Context) {
	jobID := c.Param("job_id")
	var result map[string]interface{}
	if err := agentclient.NewClient(h.agentSock).Call("antivirus.status", map[string]interface{}{
		"job_id": jobID,
	}, &result); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
