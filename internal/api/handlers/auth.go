package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type AuthHandler struct {
	users  *store.UserStore
	secret string
	expiry string
}

func NewAuthHandler(users *store.UserStore, secret, expiry string) *AuthHandler {
	return &AuthHandler{users: users, secret: secret, expiry: expiry}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByUsername(req.Username)
	if err != nil || !h.users.CheckPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if user.Status == "suspended" {
		c.JSON(http.StatusForbidden, gin.H{"error": "account suspended"})
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Role, h.secret, h.expiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	// Set the same JWT in an httpOnly cookie so browser-driven requests
	// (the FileBrowser iframe, future preview tabs) can authenticate
	// without the SPA having to inject the Bearer header. The Bearer
	// path stays the canonical one for axios; this cookie is just a
	// fallback that nginx/auth_request can reach.
	//
	// Hardening (V13):
	//  - SameSite=Strict so cross-origin POST/PUT/DELETE with the cookie
	//    can't be triggered from a malicious page (CSRF defence).
	//  - Secure=true when the request arrived over TLS so the cookie
	//    only travels on HTTPS. We detect HTTPS via the X-Forwarded-Proto
	//    header (nginx terminates TLS) or c.Request.TLS != nil (direct).
	//    On plain HTTP dev installs the cookie won't be set; the Bearer
	//    path still works because axios sends Authorization regardless.
	secure := c.Request.TLS != nil ||
		strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("zenspanel_token", token, 24*60*60, "/", "", secure, true)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":               user.ID,
			"username":         user.Username,
			"email":            user.Email,
			"role":             user.Role,
			"terminal_enabled": user.TerminalEnabled,
			"backup_enabled":   user.BackupEnabled,
			"package_id":       user.PackageID,
			"php_version":      user.PHPVersion,
		},
	})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := auth.GetUserID(c)
	user, err := h.users.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":               user.ID,
		"username":         user.Username,
		"email":            user.Email,
		"role":             user.Role,
		"terminal_enabled": user.TerminalEnabled,
		"backup_enabled":   user.BackupEnabled,
		"package_id":       user.PackageID,
		"php_version":      user.PHPVersion,
	})
}

// Impersonate mints a short-lived token for the target user on behalf of
// the calling admin. The token carries an ImpersonatedBy claim so audit
// logs can trace the session back to the admin who initiated it.
// Only admins may call this; the route is gated by RequireRole("admin").
func (h *AuthHandler) Impersonate(c *gin.Context) {
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	target, err := h.users.GetByID(targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if target.Status == "suspended" {
		c.JSON(http.StatusForbidden, gin.H{"error": "target account is suspended"})
		return
	}
	// Prevent impersonating another admin — admins should log in normally.
	if target.Role == "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot impersonate admin accounts"})
		return
	}

	adminID := auth.GetUserID(c)
	// Short TTL — 1 hour is enough for a support session; the admin can
	// re-impersonate if they need longer. Using a fixed duration rather
	// than the global expiry so a long-lived admin token doesn't produce
	// an equally long-lived impersonation token.
	token, err := auth.GenerateTokenAs(target.ID, target.Role, adminID, h.secret, "1h")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":               target.ID,
			"username":         target.Username,
			"email":            target.Email,
			"role":             target.Role,
			"terminal_enabled": target.TerminalEnabled,
			"backup_enabled":   target.BackupEnabled,
			"package_id":       target.PackageID,
			"php_version":      target.PHPVersion,
		},
	})
}

// FileBrowserAuth is the auth_request endpoint nginx hits before
// proxying /filebrowser/* to the FileBrowser service. It validates the
// JWT (via the JWTMiddleware that gates this route in router.go) and
// echoes the panel username back as X-Auth-User. nginx forwards that
// header to FileBrowser, which is configured with auth_method=proxy
// and auto-creates a sandbox under root/<username>/ on first hit.
//
// We return 200 with no body so nginx's auth_request directive treats
// it as success and reads the X-Auth-User response header. 401 here
// would short-circuit the parent request; the JWT middleware already
// does that on its own when the token is missing or invalid.
func (h *AuthHandler) FileBrowserAuth(c *gin.Context) {
	userID := auth.GetUserID(c)
	user, err := h.users.GetByID(userID)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	c.Header("X-Auth-User", user.Username)
	c.Status(http.StatusOK)
}
