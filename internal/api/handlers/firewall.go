package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	agentclient "github.com/zenspanel/zenspanel/internal/agent"
)

type FirewallHandler struct {
	agentSock string
}

func NewFirewallHandler(agentSock string) *FirewallHandler {
	return &FirewallHandler{agentSock: agentSock}
}

func (h *FirewallHandler) ListBlocked(c *gin.Context) {
	var result map[string]interface{}
	if err := agentclient.NewClient(h.agentSock).Call("firewall.list_blocked", map[string]interface{}{}, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *FirewallHandler) Block(c *gin.Context) {
	var req struct {
		IP     string `json:"ip"     binding:"required"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := agentclient.NewClient(h.agentSock).Call("firewall.block", map[string]interface{}{
		"ip":     req.IP,
		"reason": req.Reason,
	}, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "blocked"})
}

// Unblock is admin-only (V37) — route is gated by RequireRole("admin").
func (h *FirewallHandler) Unblock(c *gin.Context) {
	var req struct {
		IP string `json:"ip" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := agentclient.NewClient(h.agentSock).Call("firewall.unblock", map[string]interface{}{
		"ip": req.IP,
	}, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "unblocked"})
}

func (h *FirewallHandler) ListJails(c *gin.Context) {
	var result map[string]interface{}
	if err := agentclient.NewClient(h.agentSock).Call("fail2ban.list_jails", map[string]interface{}{}, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *FirewallHandler) SetJail(c *gin.Context) {
	name := c.Param("name")
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := agentclient.NewClient(h.agentSock).Call("fail2ban.set_jail", map[string]interface{}{
		"name":    name,
		"enabled": req.Enabled,
	}, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}
