package handlers

import (
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type SubdomainHandler struct {
	subdomains  *store.SubdomainStore
	domains     *store.DomainStore
	users       *store.UserStore
	phpVersions *store.PHPVersionStore
	agentSock   string
	homeBase    string
}

func NewSubdomainHandler(
	subdomains *store.SubdomainStore,
	domains *store.DomainStore,
	users *store.UserStore,
	phpVersions *store.PHPVersionStore,
	agentSock, homeBase string,
) *SubdomainHandler {
	return &SubdomainHandler{
		subdomains:  subdomains,
		domains:     domains,
		users:       users,
		phpVersions: phpVersions,
		agentSock:   agentSock,
		homeBase:    homeBase,
	}
}

// List returns subdomains. ?parent_id= scopes to one parent; admins see
// all subdomains across the panel; regular users see only their own.
func (h *SubdomainHandler) List(c *gin.Context) {
	role := auth.GetRole(c)
	userID := auth.GetUserID(c)

	if pid := c.Query("parent_id"); pid != "" {
		parentID, err := strconv.ParseUint(pid, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid parent_id"})
			return
		}
		// Must own the parent (or be admin) to enumerate its children.
		parent, err := h.domains.GetByID(parentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "parent domain not found"})
			return
		}
		if role == "user" && parent.UserID != userID {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		subs, err := h.subdomains.ListByParentID(parentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": subs})
		return
	}

	var subs []store.Subdomain
	var err error
	if role == "admin" {
		subs, err = h.subdomains.ListAll()
	} else {
		subs, err = h.subdomains.ListByUserID(userID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": subs})
}

func (h *SubdomainHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	sub, err := h.subdomains.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subdomain not found"})
		return
	}
	if auth.GetRole(c) == "user" && sub.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	c.JSON(http.StatusOK, sub)
}

// Create provisions a subdomain under an existing parent domain owned by
// the requester. Order of side effects: validate → check FQDN free →
// insert row → ask agent to write nginx vhost → flip status to active.
// On agent failure we delete the row so the unique constraint on `fqdn`
// doesn't permanently block retries (V10 says we keep the row only on
// the DELETE side, where leaving an orphan row would block recreate;
// CREATE-side failures should leave nothing behind).
func (h *SubdomainHandler) Create(c *gin.Context) {
	var req struct {
		ParentDomainID uint64 `json:"parent_domain_id" binding:"required"`
		Subdomain      string `json:"subdomain" binding:"required"`
		PHPVersion     string `json:"php_version"`
		DocRoot        string `json:"doc_root"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// V2 + V7
	req.Subdomain = strings.ToLower(strings.TrimSpace(req.Subdomain))
	if err := ValidateSubdomainLabel(req.Subdomain); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	parent, err := h.domains.GetByID(req.ParentDomainID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "parent domain not found"})
		return
	}
	// V1: ownership check. Admin bypass kept consistent w/ DomainHandler.
	requesterID := auth.GetUserID(c)
	if auth.GetRole(c) == "user" && parent.UserID != requesterID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	fqdn := req.Subdomain + "." + parent.Domain

	// V3: collide-check across both `domains` and `subdomains`.
	if existingSub, _ := h.subdomains.GetByFQDN(fqdn); existingSub != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "subdomain already exists"})
		return
	}
	if existingDom := lookupDomainByName(h.domains, fqdn); existingDom != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "fqdn collides with an existing parent domain"})
		return
	}

	// V9: php_version must be enabled. Fall back to the parent domain's
	// version when the request omits it — most users want subdomain-on-
	// same-runtime, and "8.3" hardcoded would surprise users on an older
	// stack.
	phpVersion := req.PHPVersion
	if phpVersion == "" {
		phpVersion = parent.PHPVersion
	}
	if !h.isPHPEnabled(phpVersion) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "php version " + phpVersion + " is not enabled"})
		return
	}

	owner := parent.UserID
	user, err := h.users.GetByID(owner)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "owner user not found"})
		return
	}

	// V4: docroot jail. Default = homeBase/<user>/public_html/<fqdn>;
	// custom paths must resolve underneath the user's home dir. We use
	// filepath.Clean rather than EvalSymlinks because the docroot may
	// not exist yet (agent creates it). Symlink-based escapes inside an
	// existing tree are still possible but require the user already
	// breached the jail — at which point bigger problems exist.
	userHome := filepath.Clean(h.homeBase + "/" + user.Username)
	docRoot := req.DocRoot
	if docRoot == "" {
		docRoot = userHome + "/public_html/" + fqdn
	} else if !filepath.IsAbs(docRoot) {
		docRoot = userHome + "/" + docRoot
	}
	docRoot = filepath.Clean(docRoot)
	if !strings.HasPrefix(docRoot+"/", userHome+"/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "doc_root must be under user home"})
		return
	}

	sub := &store.Subdomain{
		UserID:         owner,
		ParentDomainID: parent.ID,
		Subdomain:      req.Subdomain,
		FQDN:           fqdn,
		DocumentRoot:   docRoot,
		PHPVersion:     phpVersion,
		SSLType:        "none",
		Status:         "pending",
	}

	if err := h.subdomains.Create(sub); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	agentClient := agent.NewClient(h.agentSock)
	// nginx.create_vhost also chmods + chowns the docroot and seeds
	// index.html (see agent/nginx.ensureDocRoot). phpfpm.create_pool is
	// idempotent — pool already exists for the user; we call it anyway
	// so a brand-new user who's never had a vhost before still gets the
	// pool created.
	if err := agentClient.Call("phpfpm.create_pool", map[string]interface{}{
		"username":    user.Username,
		"php_version": phpVersion,
	}, nil); err != nil {
		_ = h.subdomains.Delete(sub.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "phpfpm.create_pool: " + err.Error()})
		return
	}
	if err := agentClient.Call("nginx.create_vhost", map[string]interface{}{
		"domain":      sub.FQDN,
		"username":    user.Username,
		"php_version": sub.PHPVersion,
		"doc_root":    sub.DocumentRoot,
	}, nil); err != nil {
		_ = h.subdomains.Delete(sub.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "nginx.create_vhost: " + err.Error()})
		return
	}

	if err := h.subdomains.Update(sub.ID, map[string]interface{}{"status": "active"}); err == nil {
		sub.Status = "active"
	}
	c.JSON(http.StatusCreated, sub)
}

func (h *SubdomainHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	sub, err := h.subdomains.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subdomain not found"})
		return
	}
	if auth.GetRole(c) == "user" && sub.UserID != auth.GetUserID(c) {
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
	delete(fields, "parent_domain_id")
	delete(fields, "subdomain")
	delete(fields, "fqdn")

	// V15: re-validate doc_root jail on every Update that touches the
	// field. Create-only validation is not enough — a non-admin could
	// otherwise PUT {document_root: "/etc/..."} directly. We resolve
	// the user's home dir and ensure the cleaned path stays under it,
	// then write the cleaned form back so the row never holds the
	// raw input.
	if newDocRoot, ok := fields["document_root"].(string); ok {
		owner, uerr := h.users.GetByID(sub.UserID)
		if uerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup user: " + uerr.Error()})
			return
		}
		userHome := filepath.Clean(h.homeBase + "/" + owner.Username)
		cleaned := filepath.Clean(newDocRoot)
		if !filepath.IsAbs(cleaned) {
			cleaned = filepath.Clean(userHome + "/" + cleaned)
		}
		if !strings.HasPrefix(cleaned+"/", userHome+"/") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "doc_root must be under user home"})
			return
		}
		fields["document_root"] = cleaned
	}

	if err := h.subdomains.Update(id, fields); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// PHP version change → re-issue pool + vhost (mirrors DomainHandler).
	if newPHP, ok := fields["php_version"].(string); ok && newPHP != sub.PHPVersion {
		if !h.isPHPEnabled(newPHP) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "php version " + newPHP + " is not enabled"})
			return
		}
		user, uerr := h.users.GetByID(sub.UserID)
		if uerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup user: " + uerr.Error()})
			return
		}
		agentClient := agent.NewClient(h.agentSock)
		if err := agentClient.Call("phpfpm.create_pool", map[string]interface{}{
			"username":    user.Username,
			"php_version": newPHP,
		}, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "phpfpm.create_pool: " + err.Error()})
			return
		}
		if err := agentClient.Call("nginx.create_vhost", map[string]interface{}{
			"domain":      sub.FQDN,
			"username":    user.Username,
			"php_version": newPHP,
			"doc_root":    sub.DocumentRoot,
		}, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "nginx.create_vhost: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// Delete tears down nginx vhost + SSL cert before removing the row.
// Agent failures are logged but don't block the row delete — V10 says
// keeping the orphan row would block recreate, which is worse than
// leaving an orphan .conf file the operator can clean up later.
func (h *SubdomainHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	sub, err := h.subdomains.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "subdomain not found"})
		return
	}
	if auth.GetRole(c) == "user" && sub.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	agentClient := agent.NewClient(h.agentSock)
	if err := agentClient.Call("nginx.delete_vhost", map[string]interface{}{
		"domain": sub.FQDN,
	}, nil); err != nil {
		log.Printf("nginx.delete_vhost failed for %s: %v", sub.FQDN, err)
	}
	if err := agentClient.Call("ssl.remove_cert", map[string]interface{}{
		"domain": sub.FQDN,
	}, nil); err != nil {
		log.Printf("ssl.remove_cert failed for %s: %v", sub.FQDN, err)
	}

	if err := h.subdomains.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// isPHPEnabled wraps the PHPVersionStore lookup. Inline here rather than
// in the store because the only consumer is this handler and the store
// already exposes a typed slice. Returns false on any error so a
// transient DB issue defaults to "deny" rather than allowing a version
// the operator may have just disabled.
func (h *SubdomainHandler) isPHPEnabled(version string) bool {
	versions, err := h.phpVersions.ListEnabled()
	if err != nil {
		return false
	}
	for _, v := range versions {
		if v.Version == version {
			return true
		}
	}
	return false
}

// lookupDomainByName scans all domains for a literal match. We don't
// expose this as a store method because the only caller is the FQDN
// collision check in Create; the alternative is adding a method we'd
// only use once.
func lookupDomainByName(s *store.DomainStore, name string) *store.Domain {
	all, err := s.ListAll()
	if err != nil {
		return nil
	}
	for i := range all {
		if all[i].Domain == name {
			return &all[i]
		}
	}
	return nil
}
