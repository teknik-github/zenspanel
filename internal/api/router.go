package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

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
	subdomains    *handlers.SubdomainHandler
	databases     *handlers.DatabaseHandler
	phpVersions   *handlers.PHPVersionHandler
	phpExtensions *handlers.PHPExtensionHandler
	cronJobs      *handlers.CronJobHandler
	logs          *handlers.LogHandler
	installer     *handlers.InstallerHandler
	firewall      *handlers.FirewallHandler
	antivirus     *handlers.AntivirusHandler
	backupTargets *handlers.BackupTargetHandler
	redirects     *handlers.RedirectHandler
	hotlink       *handlers.HotlinkHandler
	apiKeys       *handlers.APIKeyHandler
	auditLogs     *handlers.AuditLogHandler
	ssl           *handlers.SSLHandler
	backups       *handlers.BackupHandler
	files         *handlers.FileManagerHandler
	system        *handlers.SystemHandler
	terminal      *handlers.TerminalHandler
	ftp           *handlers.FTPHandler
	apiKeyStore   *store.APIKeyStore
	auditLogStore *store.AuditLogStore
	userStore     *store.UserStore
	redis         *redis.Client
	jwtSecret     string
	frontendDir   string // path to /opt/zenspanel/frontend
}

func NewRouter(
	authH *handlers.AuthHandler,
	usersH *handlers.UserHandler,
	packagesH *handlers.PackageHandler,
	domainsH *handlers.DomainHandler,
	subdomainsH *handlers.SubdomainHandler,
	databasesH *handlers.DatabaseHandler,
	phpVersionsH *handlers.PHPVersionHandler,
	phpExtensionsH *handlers.PHPExtensionHandler,
	cronJobsH *handlers.CronJobHandler,
	logsH *handlers.LogHandler,
	installerH *handlers.InstallerHandler,
	firewallH *handlers.FirewallHandler,
	antivirusH *handlers.AntivirusHandler,
	backupTargetsH *handlers.BackupTargetHandler,
	redirectsH     *handlers.RedirectHandler,
	hotlinkH       *handlers.HotlinkHandler,
	apiKeysH *handlers.APIKeyHandler,
	auditLogsH *handlers.AuditLogHandler,
	sslH *handlers.SSLHandler,
	backupsH *handlers.BackupHandler,
	filesH *handlers.FileManagerHandler,
	systemH *handlers.SystemHandler,
	terminalH *handlers.TerminalHandler,
	ftpH *handlers.FTPHandler,
	apiKeyStore *store.APIKeyStore,
	auditLogStore *store.AuditLogStore,
	userStore *store.UserStore,
	rdb *redis.Client,
	jwtSecret string,
	frontendDir string,
) *Router {
	return &Router{
		auth:          authH,
		users:         usersH,
		packages:      packagesH,
		domains:       domainsH,
		subdomains:    subdomainsH,
		databases:     databasesH,
		phpVersions:   phpVersionsH,
		phpExtensions: phpExtensionsH,
		cronJobs:      cronJobsH,
		logs:          logsH,
		installer:     installerH,
		firewall:      firewallH,
		antivirus:     antivirusH,
		backupTargets: backupTargetsH,
		redirects:     redirectsH,
		hotlink:       hotlinkH,
		apiKeys:       apiKeysH,
		auditLogs:     auditLogsH,
		ssl:           sslH,
		backups:       backupsH,
		files:         filesH,
		system:        systemH,
		terminal:      terminalH,
		ftp:           ftpH,
		apiKeyStore:   apiKeyStore,
		auditLogStore: auditLogStore,
		userStore:     userStore,
		redis:         rdb,
		jwtSecret:     jwtSecret,
		frontendDir:   frontendDir,
	}
}

func (r *Router) Setup() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	e := gin.New()
	e.Use(gin.Logger(), gin.Recovery())
	// Cap multipart payloads at 64 MiB so a runaway upload can't eat all
	// the API's memory before the handler's own size guard kicks in.
	// Must stay in sync with handlers.maxUploadSize and
	// agent/filemanager.maxUploadSize.
	e.MaxMultipartMemory = 64 << 20

	// public routes
	// Login is rate-limited per IP. Prefer the Redis-backed limiter when
	// Redis is available so the counter is shared across every API instance
	// behind a load balancer; fall back to the in-memory limiter for
	// single-server deployments.
	var loginLimiter gin.HandlerFunc
	if r.redis != nil {
		loginLimiter = middleware.RateLimitRedis(r.redis, 10, time.Minute)
	} else {
		loginLimiter = middleware.RateLimit(10, time.Minute)
	}
	e.POST("/api/v1/auth/login", loginLimiter, r.auth.Login)
	// 2FA verification endpoints are public — the browser can't attach a
	// JWT before completing the 2FA step. The temp_token is the credential.
	e.POST("/api/v1/auth/2fa/verify", r.auth.TOTPVerify)
	e.POST("/api/v1/auth/2fa/recover", r.auth.TOTPRecover)

	// phpMyAdmin SSO redeem — no JWT required, the URL token is the
	// credential. Lives outside the protected /api/v1 group because the
	// browser opens this in a new tab where the JWT bearer header isn't
	// sent. The token is one-time-use and expires in 60 seconds.
	e.GET("/api/v1/phpmyadmin/sso/:token", r.databases.RedeemPHPMyAdmin)

	// Terminal WebSocket — same pattern as phpMyAdmin SSO. Browsers
	// can't attach JWT headers to WebSocket handshakes from page JS, so
	// the user mints a one-time token via the JWT-protected
	// /terminal/token endpoint and passes it as a query string here.
	e.GET("/ws/terminal", r.terminal.Connect)

	// audit middleware records mutating requests on both protected and
	// external groups so the audit_logs table covers admin actions and
	// billing-system actions alike
	audit := middleware.Audit(r.auditLogStore)

	// protected routes
	api := e.Group("/api/v1", auth.JWTMiddleware(r.jwtSecret, r.userStore), audit)
	{
		api.GET("/auth/me", r.auth.Me)
		api.GET("/auth/filebrowser", r.auth.FileBrowserAuth)
		api.POST("/users/:id/impersonate", auth.RequireRole("admin"), r.auth.Impersonate)
		// 2FA management — JWT required (user must be logged in to set up/disable)
		api.POST("/auth/2fa/setup", r.auth.TOTPSetup)
		api.POST("/auth/2fa/confirm", r.auth.TOTPConfirm)
		api.DELETE("/auth/2fa", r.auth.TOTPDisable)

		// users
		api.GET("/users", auth.RequireRole("admin"), r.users.List)
		api.GET("/users/metrics", auth.RequireRole("admin"), r.users.AllMetrics)
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
		api.GET("/domains/:id/logs", r.logs.DomainLogs)
		api.POST("/domains/:id/backup", r.backups.DomainBackup)
		api.POST("/domains/:id/suspend", r.domains.SuspendDomain)
		api.POST("/domains/:id/unsuspend", r.domains.UnsuspendDomain)
		api.GET("/domains/:id/redirects", r.redirects.List)
		api.POST("/domains/:id/redirects", r.redirects.Create)
		api.PUT("/domains/:id/redirects/:rid", r.redirects.Update)
		api.DELETE("/domains/:id/redirects/:rid", r.redirects.Delete)
		api.GET("/domains/:id/hotlink", r.hotlink.Get)
		api.PUT("/domains/:id/hotlink", r.hotlink.Set)

		// subdomains
		api.GET("/subdomains", r.subdomains.List)
		api.POST("/subdomains", r.subdomains.Create)
		api.GET("/subdomains/:id", r.subdomains.Get)
		api.PUT("/subdomains/:id", r.subdomains.Update)
		api.DELETE("/subdomains/:id", r.subdomains.Delete)
		api.POST("/subdomains/:id/ssl", r.ssl.IssueForSubdomain)
		api.DELETE("/subdomains/:id/ssl", r.ssl.RemoveForSubdomain)

		// databases
		api.GET("/databases", r.databases.List)
		api.POST("/databases", r.databases.Create)
		api.DELETE("/databases/:id", r.databases.Delete)
		api.POST("/databases/:id/reset-password", r.databases.ResetPassword)
		api.GET("/databases/:id/phpmyadmin", r.databases.GetPHPMyAdminToken)
		// LaunchPHPMyAdmin: resets the DB user password, mints a one-time
		// SSO token in Redis, returns the redeem URL. Frontend opens that
		// URL in a new tab; the redeem endpoint (registered above outside
		// the JWT group) serves a self-submitting login form.
		api.GET("/databases/:id/phpmyadmin/launch", r.databases.LaunchPHPMyAdmin)

		// php versions
		api.GET("/php-versions", r.phpVersions.List)
		api.GET("/php-versions/enabled", r.phpVersions.ListEnabled)
		api.PUT("/php-versions/:id/enable", auth.RequireRole("admin"), r.phpVersions.Enable)
		api.PUT("/php-versions/:id/disable", auth.RequireRole("admin"), r.phpVersions.Disable)

		// php extensions — admin manages global catalog, users toggle per-user overrides
		api.GET("/admin/php-extensions", auth.RequireRole("admin"), r.phpExtensions.AdminList)
		api.PUT("/admin/php-extensions/:id", auth.RequireRole("admin"), r.phpExtensions.AdminUpdate)
		api.POST("/admin/php-extensions/seed", auth.RequireRole("admin"), r.phpExtensions.AdminSeed)

		// backup targets — admin manages S3/remote destinations
		api.GET("/admin/backup-targets", auth.RequireRole("admin"), r.backupTargets.List)
		api.POST("/admin/backup-targets", auth.RequireRole("admin"), r.backupTargets.Create)
		api.PUT("/admin/backup-targets/:id", auth.RequireRole("admin"), r.backupTargets.Update)
		api.DELETE("/admin/backup-targets/:id", auth.RequireRole("admin"), r.backupTargets.Delete)
		api.POST("/admin/backup-targets/:id/test", auth.RequireRole("admin"), r.backupTargets.Test)
		api.GET("/php-extensions", r.phpExtensions.UserList)
		api.PUT("/php-extensions", r.phpExtensions.UserUpdate)

		// cron jobs
		api.GET("/cron-jobs", r.cronJobs.List)
		api.POST("/cron-jobs", r.cronJobs.Create)
		api.PUT("/cron-jobs/:id", r.cronJobs.Update)
		api.DELETE("/cron-jobs/:id", r.cronJobs.Delete)

		// website installer
		api.GET("/installer/apps", r.installer.ListApps)
		api.POST("/installer/install", r.installer.Install)
		api.GET("/installer/status/:job_id", r.installer.Status)

		// antivirus — user scans their own home directory (V40)
		api.GET("/antivirus/status", r.antivirus.DaemonStatus)
		api.POST("/antivirus/scan", r.antivirus.Scan)
		api.GET("/antivirus/scan/:job_id", r.antivirus.ScanStatus)
		api.GET("/antivirus/alerts", r.antivirus.Alerts)
		api.GET("/antivirus/poll", r.antivirus.PollAlerts)
		api.POST("/antivirus/watch", r.antivirus.WatchStart)
		api.DELETE("/antivirus/watch/:watch_id", r.antivirus.WatchStop)

		// firewall — all routes admin-only (V37)
		api.GET("/admin/firewall/blocked", auth.RequireRole("admin"), r.firewall.ListBlocked)
		api.POST("/admin/firewall/block", auth.RequireRole("admin"), r.firewall.Block)
		api.POST("/admin/firewall/unblock", auth.RequireRole("admin"), r.firewall.Unblock)
		api.GET("/admin/firewall/fail2ban/jails", auth.RequireRole("admin"), r.firewall.ListJails)
		api.PUT("/admin/firewall/fail2ban/jails/:name", auth.RequireRole("admin"), r.firewall.SetJail)

		// api keys
		api.GET("/api-keys", auth.RequireRole("admin"), r.apiKeys.List)
		api.POST("/api-keys", auth.RequireRole("admin"), r.apiKeys.Create)
		api.DELETE("/api-keys/:id", auth.RequireRole("admin"), r.apiKeys.Revoke)

		// audit logs
		api.GET("/audit-logs", auth.RequireRole("admin"), r.auditLogs.List)

		// system stats — admin dashboard host metrics
		api.GET("/system/stats", auth.RequireRole("admin"), r.system.Stats)
		api.GET("/system/version", auth.RequireRole("admin"), r.system.Version)
		api.GET("/system/update/check", auth.RequireRole("admin"), r.system.CheckUpdate)
		api.POST("/system/update/run", auth.RequireRole("admin"), r.system.RunUpdate)
		api.GET("/system/update/status", auth.RequireRole("admin"), r.system.UpdateStatus)
		api.POST("/system/maintenance", auth.RequireRole("admin"), r.system.Maintenance)

		// backups (per-user)
		api.GET("/backups", r.backups.List)
		api.POST("/backups", r.backups.Create)
		api.GET("/backups/:id", r.backups.Get)
		api.GET("/backups/:id/download", r.backups.Download)
		api.POST("/backups/:id/restore", r.backups.Restore)
		api.DELETE("/backups/:id", r.backups.Delete)

		// file manager (per-user, scoped to caller's home directory)
		api.GET("/files", r.files.List)
		api.GET("/files/content", r.files.Read)
		api.POST("/files/content", r.files.Write)
		api.POST("/files/mkdir", r.files.Mkdir)
		api.PUT("/files/rename", r.files.Rename)
		api.DELETE("/files", r.files.Delete)
		api.POST("/files/upload", r.files.Upload)
		api.PUT("/files/chmod", r.files.Chmod)
		api.POST("/files/copy", r.files.Copy)
		api.POST("/files/compress", r.files.Compress)
		api.POST("/files/extract", r.files.Extract)

		// ftp accounts
		api.GET("/ftp", r.ftp.List)
		api.POST("/ftp", r.ftp.Create)
		api.DELETE("/ftp/:id", r.ftp.Delete)

		// terminal — token endpoint is JWT-gated; the WS endpoint that
		// redeems the token is registered above outside this group.
		api.POST("/terminal/token", r.terminal.GetToken)
		api.POST("/admin/terminal/token", auth.RequireRole("admin"), r.terminal.AdminGetToken)
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

	// Serve frontend SPAs when frontendDir is configured. Allows direct
	// port access without nginx. nginx is still preferred for production.
	if r.frontendDir != "" {
		adminDir := filepath.Join(r.frontendDir, "admin")
		userDir := filepath.Join(r.frontendDir, "user")
		if _, err := os.Stat(adminDir); err == nil {
			e.GET("/admin", func(c *gin.Context) {
				c.Redirect(http.StatusMovedPermanently, "/admin/")
			})
			e.GET("/admin/*path", spaHandler(adminDir, "/admin"))
		}
		if _, err := os.Stat(userDir); err == nil {
			// User SPA catch-all — skip API/WS/service paths.
			e.NoRoute(func(c *gin.Context) {
				p := c.Request.URL.Path
				if strings.HasPrefix(p, "/api") || strings.HasPrefix(p, "/ws") ||
					strings.HasPrefix(p, "/phpmyadmin") || strings.HasPrefix(p, "/filebrowser") ||
					strings.HasPrefix(p, "/admin") {
					c.Status(http.StatusNotFound)
					return
				}
				spaHandler(userDir, "")(c)
			})
		}
	}

	return e
}

// spaHandler serves a Vue SPA from dir with SPA fallback to index.html.
// stripPrefix is removed from the URL path before looking up files
// (e.g. "/admin/" so /admin/updates → /updates inside the admin dist dir).
func spaHandler(dir, stripPrefix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Strip the mount prefix to get the file path within the dist dir.
		urlPath := strings.TrimPrefix(c.Request.URL.Path, stripPrefix)
		if urlPath == "" {
			urlPath = "/"
		}
		fullPath := filepath.Join(dir, filepath.Clean("/"+urlPath))

		// If the file exists, serve it directly (assets, JS, CSS).
		if _, err := os.Stat(fullPath); err == nil {
			http.ServeFile(c.Writer, c.Request, fullPath)
			return
		}
		// Otherwise serve index.html — the SPA router handles the route.
		http.ServeFile(c.Writer, c.Request, filepath.Join(dir, "index.html"))
	}
}
