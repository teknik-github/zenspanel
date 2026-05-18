package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zenspanel/zenspanel/internal/api/handlers"
	"github.com/zenspanel/zenspanel/internal/api/middleware"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)
type Router struct {
	auth          *handlers.AuthHandler
	users         *handlers.UserHandler
	packages      *handlers.PackageHandler
	domains       *handlers.DomainHandler
	databases     *handlers.DatabaseHandler
	phpVersions   *handlers.PHPVersionHandler
	apiKeys       *handlers.APIKeyHandler
	auditLogs     *handlers.AuditLogHandler
	ssl           *handlers.SSLHandler
	backups       *handlers.BackupHandler
	apiKeyStore   *store.APIKeyStore
	auditLogStore *store.AuditLogStore
	jwtSecret     string
}

func NewRouter(
	authH *handlers.AuthHandler,
	usersH *handlers.UserHandler,
	packagesH *handlers.PackageHandler,
	domainsH *handlers.DomainHandler,
	databasesH *handlers.DatabaseHandler,
	phpVersionsH *handlers.PHPVersionHandler,
	apiKeysH *handlers.APIKeyHandler,
	auditLogsH *handlers.AuditLogHandler,
	sslH *handlers.SSLHandler,
	backupsH *handlers.BackupHandler,
	apiKeyStore *store.APIKeyStore,
	auditLogStore *store.AuditLogStore,
	jwtSecret string,
) *Router {
	return &Router{
		auth:          authH,
		users:         usersH,
		packages:      packagesH,
		domains:       domainsH,
		databases:     databasesH,
		phpVersions:   phpVersionsH,
		apiKeys:       apiKeysH,
		auditLogs:     auditLogsH,
		ssl:           sslH,
		backups:       backupsH,
		apiKeyStore:   apiKeyStore,
		auditLogStore: auditLogStore,
		jwtSecret:     jwtSecret,
	}
}

func (r *Router) Setup() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	e := gin.New()
	e.Use(gin.Logger(), gin.Recovery())

	// public routes
	// Login is rate-limited per IP to make brute-forcing usernames painful
	// without locking out legitimate users on shared NATs (10/min is enough
	// headroom for a typo-prone admin and well below what any sensible
	// attacker needs to make progress).
	e.POST("/api/v1/auth/login", middleware.RateLimit(10, time.Minute), r.auth.Login)

	// audit middleware records mutating requests on both protected and
	// external groups so the audit_logs table covers admin actions and
	// billing-system actions alike
	audit := middleware.Audit(r.auditLogStore)

	// protected routes
	api := e.Group("/api/v1", auth.JWTMiddleware(r.jwtSecret), audit)
	{
		api.GET("/auth/me", r.auth.Me)

		// users
		api.GET("/users", auth.RequireRole("admin"), r.users.List)
		api.GET("/users/:id", r.users.Get)
		api.POST("/users", auth.RequireRole("admin"), r.users.Create)
		api.PUT("/users/:id", auth.RequireRole("admin"), r.users.Update)
		api.DELETE("/users/:id", auth.RequireRole("admin"), r.users.Delete)
		api.PUT("/users/:id/suspend", auth.RequireRole("admin"), r.users.Suspend)
		api.PUT("/users/:id/unsuspend", auth.RequireRole("admin"), r.users.Unsuspend)
		api.PUT("/users/:id/package", auth.RequireRole("admin"), r.users.ChangePackage)
		api.GET("/users/:id/usage", r.users.GetUsage)

		// packages
		api.GET("/packages", r.packages.List)
		api.GET("/packages/:id", r.packages.Get)
		api.POST("/packages", auth.RequireRole("admin"), r.packages.Create)
		api.PUT("/packages/:id", auth.RequireRole("admin"), r.packages.Update)
		api.DELETE("/packages/:id", auth.RequireRole("admin"), r.packages.Delete)

		// domains
		api.GET("/domains", r.domains.List)
		api.GET("/domains/:id", r.domains.Get)
		api.POST("/domains", r.domains.Create)
		api.PUT("/domains/:id", r.domains.Update)
		api.DELETE("/domains/:id", r.domains.Delete)

		// ssl (per-domain)
		api.POST("/domains/:id/ssl", r.ssl.Issue)
		api.DELETE("/domains/:id/ssl", r.ssl.Remove)

		// databases
		api.GET("/databases", r.databases.List)
		api.POST("/databases", r.databases.Create)
		api.DELETE("/databases/:id", r.databases.Delete)
		api.GET("/databases/:id/phpmyadmin", r.databases.GetPHPMyAdminToken)

		// php versions
		api.GET("/php-versions", r.phpVersions.List)
		api.GET("/php-versions/enabled", r.phpVersions.ListEnabled)
		api.PUT("/php-versions/:id/enable", auth.RequireRole("admin"), r.phpVersions.Enable)
		api.PUT("/php-versions/:id/disable", auth.RequireRole("admin"), r.phpVersions.Disable)

		// api keys
		api.GET("/api-keys", auth.RequireRole("admin"), r.apiKeys.List)
		api.POST("/api-keys", auth.RequireRole("admin"), r.apiKeys.Create)
		api.DELETE("/api-keys/:id", auth.RequireRole("admin"), r.apiKeys.Revoke)

		// audit logs
		api.GET("/audit-logs", auth.RequireRole("admin"), r.auditLogs.List)

		// backups (per-user)
		api.GET("/backups", r.backups.List)
		api.POST("/backups", r.backups.Create)
		api.GET("/backups/:id/download", r.backups.Download)
		api.POST("/backups/:id/restore", r.backups.Restore)
		api.DELETE("/backups/:id", r.backups.Delete)
	}

	// External API — authenticated via X-API-Key header. The endpoints
	// here are the subset a billing system (WHMCS, FOSSBilling, custom)
	// would call to provision and manage hosting accounts. Each route
	// requires the matching permission string on the API key, so an
	// integration that only needs read access can be issued a key without
	// suspend/create rights.
	ext := e.Group("/api/v1/external", auth.APIKeyMiddleware(r.apiKeyStore), audit)
	{
		ext.GET("/users", auth.RequirePermission("read_user"), r.users.List)
		ext.GET("/users/:id", auth.RequirePermission("read_user"), r.users.Get)
		ext.POST("/users", auth.RequirePermission("create_user"), r.users.Create)
		ext.PUT("/users/:id/suspend", auth.RequirePermission("suspend_user"), r.users.Suspend)
		ext.PUT("/users/:id/unsuspend", auth.RequirePermission("suspend_user"), r.users.Unsuspend)
		ext.PUT("/users/:id/package", auth.RequirePermission("change_package"), r.users.ChangePackage)
		ext.GET("/users/:id/usage", auth.RequirePermission("read_user"), r.users.GetUsage)

		ext.GET("/packages", auth.RequirePermission("read_package"), r.packages.List)
		ext.GET("/packages/:id", auth.RequirePermission("read_package"), r.packages.Get)
	}

	return e
}
