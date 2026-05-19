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
	domains    *store.DomainStore
	subdomains *store.SubdomainStore
	users      *store.UserStore
	agentSock  string
	homeBase   string
}

func NewDomainHandler(
	domains *store.DomainStore,
	subdomains *store.SubdomainStore,
	users *store.UserStore,
	agentSock, homeBase string,
) *DomainHandler {
	return &DomainHandler{
		domains:    domains,
		subdomains: subdomains,
		users:      users,
		agentSock:  agentSock,
		homeBase:   homeBase,
	}
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
	if auth.GetRole(c) == "user" && domain.UserID != auth.GetUserID(c) {
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
	if auth.GetRole(c) == "user" && domain.UserID != auth.GetUserID(c) {
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

	// If PHP version changed, the user's FPM pool for the new version must
	// exist and the nginx vhost has to be regenerated to point its
	// fastcgi_pass at the right socket. The DB update alone changes nothing
	// the kernel can see — without these calls, the site stays on the old
	// PHP until the next restart of the agent or a manual reload.
	//
	// Errors here used to be swallowed with `_` — meaning the row got
	// updated to the new version even when the new pool couldn't be
	// created or php<ver>-fpm wasn't running, which left the site with
	// nginx pointing at a non-existent socket and ERR_CONNECTION_REFUSED
	// in the browser. Now we collect them and report 500 if any fail
	// so the operator knows to investigate.
	if newPHP, ok := fields["php_version"].(string); ok && newPHP != domain.PHPVersion {
		user, uerr := h.users.GetByID(domain.UserID)
		if uerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "lookup user for php switch: " + uerr.Error(),
			})
			return
		}
		agentClient := agent.NewClient(h.agentSock)
		// CreatePool now also enables+starts the php<ver>-fpm service
		// (Ubuntu only auto-starts the version installed first), so a
		// switch to PHP 8.2 from 8.3 doesn't leave the 8.2 unit dead.
		if err := agentClient.Call("phpfpm.create_pool", map[string]interface{}{
			"username":    user.Username,
			"php_version": newPHP,
		}, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "phpfpm.create_pool for " + newPHP + ": " + err.Error(),
			})
			return
		}
		// Rewrite the vhost so fastcgi_pass points at the new socket.
		if err := agentClient.Call("nginx.create_vhost", map[string]interface{}{
			"domain":      domain.Domain,
			"username":    user.Username,
			"php_version": newPHP,
			"doc_root":    domain.DocumentRoot,
		}, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "nginx.create_vhost for " + newPHP + ": " + err.Error(),
			})
			return
		}
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
	if auth.GetRole(c) == "user" && domain.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	agentClient := agent.NewClient(h.agentSock)

	// Tear down every subdomain that hangs off this parent first. The
	// FK is ON DELETE CASCADE so the rows would disappear anyway when
	// we drop the parent — but the kernel knows nothing about the
	// nginx .conf files, and a CASCADE'd row leaves the .conf orphaned
	// (nginx reload then errors on the next change). Walk subdomains
	// explicitly, ask the agent to remove vhost + SSL per child, then
	// let the FK handle the row delete.
	if h.subdomains != nil {
		subs, _ := h.subdomains.ListByParentID(id)
		for _, s := range subs {
			if err := agentClient.Call("nginx.delete_vhost", map[string]interface{}{
				"domain": s.FQDN,
			}, nil); err != nil {
				log.Printf("nginx.delete_vhost subdomain %s: %v", s.FQDN, err)
			}
			if err := agentClient.Call("ssl.remove_cert", map[string]interface{}{
				"domain": s.FQDN,
			}, nil); err != nil {
				log.Printf("ssl.remove_cert subdomain %s: %v", s.FQDN, err)
			}
		}
	}

	// Ask the agent to remove the nginx vhost first. If it fails we still
	// want to delete the panel row — leaving an orphan row is worse than
	// leaving an orphan .conf because the row blocks recreate.
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
