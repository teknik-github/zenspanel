package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	agentnginx "github.com/zenspanel/zenspanel/agent/nginx"
	agentclient "github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type RedirectHandler struct {
	redirects *store.RedirectStore
	domains   *store.DomainStore
	agentSock string
}

func NewRedirectHandler(redirects *store.RedirectStore, domains *store.DomainStore, agentSock string) *RedirectHandler {
	return &RedirectHandler{redirects: redirects, domains: domains, agentSock: agentSock}
}

// syncToNginx rebuilds the redirect snippet for a domain from DB (V55).
func (h *RedirectHandler) syncToNginx(domainID uint64, domainName string) error {
	rows, err := h.redirects.ListByDomainID(domainID)
	if err != nil {
		return err
	}
	nginxRedirects := make([]agentnginx.Redirect, len(rows))
	for i, r := range rows {
		nginxRedirects[i] = agentnginx.Redirect{
			SourcePath: r.SourcePath,
			DestURL:    r.DestURL,
			Type:       r.Type,
			Enabled:    r.Enabled,
		}
	}
	return agentclient.NewClient(h.agentSock).Call("nginx.sync_redirects", map[string]interface{}{
		"domain":    domainName,
		"redirects": nginxRedirects,
	}, nil)
}

func (h *RedirectHandler) List(c *gin.Context) {
	domainID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	domain, err := h.domains.GetByID(domainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	// Ownership check (V53).
	if auth.GetRole(c) == "user" && domain.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	rows, err := h.redirects.ListByDomainID(domainID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows})
}

func (h *RedirectHandler) Create(c *gin.Context) {
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
		SourcePath string `json:"source_path" binding:"required"`
		DestURL    string `json:"dest_url"    binding:"required"`
		Type       string `json:"type"`
		Enabled    *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Type != "301" && req.Type != "302" {
		req.Type = "301"
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	r := &store.DomainRedirect{
		DomainID:   domainID,
		SourcePath: req.SourcePath,
		DestURL:    req.DestURL,
		Type:       req.Type,
		Enabled:    enabled,
	}
	if err := h.redirects.Create(r); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.syncToNginx(domainID, domain.Domain)
	c.JSON(http.StatusCreated, r)
}

func (h *RedirectHandler) Update(c *gin.Context) {
	domainID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	rid, _ := strconv.ParseUint(c.Param("rid"), 10, 64)

	domain, err := h.domains.GetByID(domainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	if auth.GetRole(c) == "user" && domain.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	existing, err := h.redirects.GetByID(rid)
	if err != nil || existing.DomainID != domainID {
		c.JSON(http.StatusNotFound, gin.H{"error": "redirect not found"})
		return
	}

	var req struct {
		SourcePath string `json:"source_path"`
		DestURL    string `json:"dest_url"`
		Type       string `json:"type"`
		Enabled    *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.SourcePath != "" {
		existing.SourcePath = req.SourcePath
	}
	if req.DestURL != "" {
		existing.DestURL = req.DestURL
	}
	if req.Type == "301" || req.Type == "302" {
		existing.Type = req.Type
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}

	if err := h.redirects.Update(rid, existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.syncToNginx(domainID, domain.Domain)
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *RedirectHandler) Delete(c *gin.Context) {
	domainID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	rid, _ := strconv.ParseUint(c.Param("rid"), 10, 64)

	domain, err := h.domains.GetByID(domainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	if auth.GetRole(c) == "user" && domain.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	existing, err := h.redirects.GetByID(rid)
	if err != nil || existing.DomainID != domainID {
		c.JSON(http.StatusNotFound, gin.H{"error": "redirect not found"})
		return
	}

	if err := h.redirects.Delete(rid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = h.syncToNginx(domainID, domain.Domain)
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
