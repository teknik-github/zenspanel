package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	agentclient "github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type AntivirusHandler struct {
	users     *store.UserStore
	alerts    *store.AntivirusAlertStore
	agentSock string
}

func NewAntivirusHandler(users *store.UserStore, alerts *store.AntivirusAlertStore, agentSock string) *AntivirusHandler {
	return &AntivirusHandler{users: users, alerts: alerts, agentSock: agentSock}
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

// Alerts returns stored realtime detection alerts for the calling user.
func (h *AntivirusHandler) Alerts(c *gin.Context) {
	userID := auth.GetUserID(c)
	alerts, err := h.alerts.ListByUserID(userID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": alerts})
}

// PollAlerts polls the agent for new realtime alerts, persists them to DB,
// and returns them. Called by the frontend on a short interval (V46).
func (h *AntivirusHandler) PollAlerts(c *gin.Context) {
	userID := auth.GetUserID(c)
	user, err := h.users.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup user"})
		return
	}

	var result struct {
		Alerts []struct {
			Path       string    `json:"path"`
			Threat     string    `json:"threat"`
			DetectedAt time.Time `json:"detected_at"`
		} `json:"alerts"`
	}
	if err := agentclient.NewClient(h.agentSock).Call("antivirus.poll_alerts", map[string]interface{}{
		"username": user.Username,
	}, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Persist new alerts to DB.
	for _, a := range result.Alerts {
		_ = h.alerts.Create(&store.AntivirusAlert{
			UserID:     userID,
			Path:       a.Path,
			Threat:     a.Threat,
			DetectedAt: a.DetectedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"new_alerts": len(result.Alerts), "alerts": result.Alerts})
}

// WatchStart starts realtime monitoring for the calling user.
func (h *AntivirusHandler) WatchStart(c *gin.Context) {
	userID := auth.GetUserID(c)
	user, err := h.users.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup user"})
		return
	}

	var result map[string]interface{}
	if err := agentclient.NewClient(h.agentSock).Call("antivirus.watch_start", map[string]interface{}{
		"username": user.Username,
	}, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// WatchStop stops realtime monitoring.
func (h *AntivirusHandler) WatchStop(c *gin.Context) {
	watchID := c.Param("watch_id")
	if err := agentclient.NewClient(h.agentSock).Call("antivirus.watch_stop", map[string]interface{}{
		"watch_id": watchID,
	}, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "stopped"})
}
