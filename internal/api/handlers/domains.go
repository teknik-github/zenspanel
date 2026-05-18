package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type DomainHandler struct {
	domains   *store.DomainStore
	users     *store.UserStore
	agentSock string
	homeBase  string
}

func NewDomainHandler(domains *store.DomainStore, users *store.UserStore, agentSock, homeBase string) *DomainHandler {
	return &DomainHandler{domains: domains, users: users, agentSock: agentSock, homeBase: homeBase}
}

func (h *DomainHandler) List(c *gin.Context) {
	role := auth.GetRole(c)
	userID := auth.GetUserID(c)

	var domains []store.Domain
	var err error
	if role == "admin" {
		domains, err = h.domains.ListAll()
	} else {
		domains, err = h.domains.ListByUserID(userID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": domains})
}

func (h *DomainHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	domain, err := h.domains.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	if auth.GetRole(c) != "admin" && domain.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	c.JSON(http.StatusOK, domain)
}

func (h *DomainHandler) Create(c *gin.Context) {
	var req struct {
		Domain     string `json:"domain" binding:"required"`
		PHPVersion string `json:"php_version"`
		UserID     uint64 `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ValidateDomain(req.Domain); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := auth.GetUserID(c)
	if auth.GetRole(c) == "admin" && req.UserID > 0 {
		userID = req.UserID
	}

	phpVersion := req.PHPVersion
	if phpVersion == "" {
		phpVersion = "8.3"
	}

	user, err := h.users.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	domain := &store.Domain{
		UserID:       userID,
		Domain:       req.Domain,
		DocumentRoot: h.homeBase + "/" + user.Username + "/public_html/" + req.Domain,
		PHPVersion:   phpVersion,
		SSLType:      "none",
		Status:       "pending",
	}

	if err := h.domains.Create(domain); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Provision the nginx vhost. On agent failure we roll back the row so
	// the unique constraint on `domain` doesn't permanently block retries.
	agentClient := agent.NewClient(h.agentSock)
	if err := agentClient.Call("nginx.create_vhost", map[string]interface{}{
		"domain":      domain.Domain,
		"username":    user.Username,
		"php_version": domain.PHPVersion,
		"doc_root":    domain.DocumentRoot,
	}, nil); err != nil {
		_ = h.domains.Delete(domain.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "provision nginx vhost: " + err.Error()})
		return
	}

	if err := h.domains.Update(domain.ID, map[string]interface{}{"status": "active"}); err == nil {
		domain.Status = "active"
	}

	c.JSON(http.StatusCreated, domain)
}

func (h *DomainHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	domain, err := h.domains.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	if auth.GetRole(c) != "admin" && domain.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	var fields map[string]interface{}
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	delete(fields, "id")
	delete(fields, "user_id")
	if err := h.domains.Update(id, fields); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *DomainHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	domain, err := h.domains.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	if auth.GetRole(c) != "admin" && domain.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	// Ask the agent to remove the nginx vhost first. If it fails we still
	// want to delete the panel row — leaving an orphan row is worse than
	// leaving an orphan .conf because the row blocks recreate.
	agentClient := agent.NewClient(h.agentSock)
	if err := agentClient.Call("nginx.delete_vhost", map[string]interface{}{
		"domain": domain.Domain,
	}, nil); err != nil {
		log.Printf("nginx.delete_vhost failed for %s: %v", domain.Domain, err)
	}

	if err := h.domains.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
