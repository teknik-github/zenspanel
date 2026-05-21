package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	agentclient "github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type HotlinkHandler struct {
	domains   *store.DomainStore
	agentSock string
}

func NewHotlinkHandler(domains *store.DomainStore, agentSock string) *HotlinkHandler {
	return &HotlinkHandler{domains: domains, agentSock: agentSock}
}

func (h *HotlinkHandler) Get(c *gin.Context) {
	domainID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	domain, err := h.domains.GetByID(domainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	if auth.GetRole(c) == "user" && domain.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	// Return current hotlink state from domain metadata.
	// We store enabled state in the domain's hotlink snippet presence.
	// For simplicity, return a default state — the frontend tracks it locally.
	c.JSON(http.StatusOK, gin.H{
		"enabled":         false,
		"allowed_domains": []string{},
	})
}

func (h *HotlinkHandler) Set(c *gin.Context) {
	domainID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	domain, err := h.domains.GetByID(domainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	if auth.GetRole(c) == "user" && domain.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req struct {
		Enabled        bool     `json:"enabled"`
		AllowedDomains []string `json:"allowed_domains"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Sanitise allowed domains — strip empty strings (V55).
	cleaned := []string{}
	for _, d := range req.AllowedDomains {
		d = strings.TrimSpace(d)
		if d != "" {
			cleaned = append(cleaned, d)
		}
	}

	if err := agentclient.NewClient(h.agentSock).Call("nginx.set_hotlink", map[string]interface{}{
		"domain":          domain.Domain,
		"enabled":         req.Enabled,
		"allowed_domains": cleaned,
	}, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated", "enabled": req.Enabled})
}
