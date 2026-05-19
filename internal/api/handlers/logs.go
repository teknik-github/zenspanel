package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	agentclient "github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type LogHandler struct {
	domains *store.DomainStore
	users   *store.UserStore
	agentSock string
}

func NewLogHandler(domains *store.DomainStore, users *store.UserStore, agentSock string) *LogHandler {
	return &LogHandler{domains: domains, users: users, agentSock: agentSock}
}

// DomainLogs returns the last N lines of the nginx access/error log or
// the PHP-FPM log for a domain. The agent enforces the path jail (V30)
// and line cap (V31) — the handler only resolves the log path from the
// domain row and passes it through.
func (h *LogHandler) DomainLogs(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain id"})
		return
	}

	domain, err := h.domains.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	// Ownership check — users can only read logs for their own domains.
	if auth.GetRole(c) == "user" && domain.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	logType := c.DefaultQuery("type", "nginx")
	lines := 100
	if l := c.Query("lines"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			lines = n
		}
	}

	var logPath string
	switch logType {
	case "nginx-access":
		logPath = fmt.Sprintf("/var/log/nginx/%s.access.log", domain.Domain)
	case "nginx-error", "nginx":
		logPath = fmt.Sprintf("/var/log/nginx/%s.error.log", domain.Domain)
	case "fpm":
		logPath = fmt.Sprintf("/var/log/php%s-fpm.log", domain.PHPVersion)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be nginx, nginx-access, nginx-error, or fpm"})
		return
	}

	var result struct {
		Lines []string `json:"lines"`
	}
	if err := agentclient.NewClient(h.agentSock).Call("logs.tail", map[string]interface{}{
		"log_path": logPath,
		"lines":    lines,
	}, &result); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"domain":   domain.Domain,
		"type":     logType,
		"log_path": logPath,
		"lines":    result.Lines,
	})
}
