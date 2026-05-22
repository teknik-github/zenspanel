package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zenspanel/zenspanel/internal/store"
)

const (
	ContextKeyUserID            = "user_id"
	ContextKeyRole              = "role"
	ContextKeyAPIKeyID          = "api_key_id"
	ContextKeyAPIKeyPermissions = "api_key_permissions"
)

func JWTMiddleware(secret string, users *store.UserStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string
		if header := c.GetHeader("Authorization"); strings.HasPrefix(header, "Bearer ") {
			tokenStr = strings.TrimPrefix(header, "Bearer ")
		} else if cookie, err := c.Cookie("zenspanel_token"); err == nil && cookie != "" {
			tokenStr = cookie
		}
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		claims, err := ValidateToken(tokenStr, secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}
		// Validate token_version against DB — BumpTokenVersion on suspend
		// increments the DB value, making all previously issued tokens stale.
		// Only check for user/admin roles (not api_key which has no version).
		// Fail-open on DB error so a migration lag or DB hiccup doesn't lock
		// out every user — the suspend revocation is best-effort in that case.
		if users != nil && (claims.Role == "user" || claims.Role == "admin") {
			dbVersion, err := users.GetTokenVersion(claims.UserID)
			if err == nil && claims.TokenVersion < dbVersion {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session revoked"})
				return
			}
		}
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyRole, claims.Role)
		c.Next()
	}
}

// APIKeyMiddleware authenticates external callers (billing systems, custom
// integrations) via the X-API-Key header. On success the request runs with
// role="api_key" so RequireRole("admin") still blocks it — admin-only
// routes are deliberately not exposed under /api/v1/external.
func APIKeyMiddleware(keys *store.APIKeyStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("X-API-Key")
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing api key"})
			return
		}
		key, err := keys.ValidateKey(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		c.Set(ContextKeyAPIKeyID, key.ID)
		c.Set(ContextKeyAPIKeyPermissions, key.Permissions)
		c.Set(ContextKeyRole, "api_key")
		c.Next()
	}
}

// RequirePermission checks the comma-separated permissions string stored on
// the API key. JWT-authenticated callers (admin/user) bypass this check —
// they're already gated by RequireRole. Use only on /api/v1/external routes.
func RequirePermission(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get(ContextKeyRole)
		if role != "api_key" {
			c.Next()
			return
		}
		permsRaw, _ := c.Get(ContextKeyAPIKeyPermissions)
		perms, _ := permsRaw.(string)
		for _, p := range strings.Split(perms, ",") {
			if strings.TrimSpace(p) == perm {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "api key missing permission: " + perm})
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get(ContextKeyRole)
		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
	}
}

func GetUserID(c *gin.Context) uint64 {
	v, _ := c.Get(ContextKeyUserID)
	id, _ := v.(uint64)
	return id
}

func GetRole(c *gin.Context) string {
	v, _ := c.Get(ContextKeyRole)
	role, _ := v.(string)
	return role
}
