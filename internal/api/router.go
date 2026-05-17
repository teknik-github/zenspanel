package api

import (
	"github.com/gin-gonic/gin"
	"github.com/zenspanel/zenspanel/internal/api/handlers"
	"github.com/zenspanel/zenspanel/internal/auth"
)

type Router struct {
	auth        *handlers.AuthHandler
	users       *handlers.UserHandler
	packages    *handlers.PackageHandler
	domains     *handlers.DomainHandler
	databases   *handlers.DatabaseHandler
	phpVersions *handlers.PHPVersionHandler
	apiKeys     *handlers.APIKeyHandler
	auditLogs   *handlers.AuditLogHandler
	jwtSecret   string
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
	jwtSecret string,
) *Router {
	return &Router{
		auth:        authH,
		users:       usersH,
		packages:    packagesH,
		domains:     domainsH,
		databases:   databasesH,
		phpVersions: phpVersionsH,
		apiKeys:     apiKeysH,
		auditLogs:   auditLogsH,
		jwtSecret:   jwtSecret,
	}
}

func (r *Router) Setup() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	e := gin.New()
	e.Use(gin.Logger(), gin.Recovery())

	// public routes
	e.POST("/api/v1/auth/login", r.auth.Login)

	// protected routes
	api := e.Group("/api/v1", auth.JWTMiddleware(r.jwtSecret))
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
	}

	return e
}
