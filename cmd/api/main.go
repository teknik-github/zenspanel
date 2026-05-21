package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/zenspanel/zenspanel/internal/api"
	"github.com/zenspanel/zenspanel/internal/api/handlers"
	"github.com/zenspanel/zenspanel/internal/config"
	"github.com/zenspanel/zenspanel/internal/store"
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z".
// Falls back to "dev" when running from source without a tag.
var version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	db, err := store.New(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := store.RunMigrations(db, "migrations"); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Try Redis. We fall back to the in-memory rate limiter if it isn't
	// available — locking out every login during a Redis outage would be a
	// worse failure mode than briefly relaxing rate limiting.
	var rdb *redis.Client
	if cfg.Redis.Addr != "" {
		client := redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := client.Ping(ctx).Err(); err != nil {
			log.Printf("WARN: Redis unavailable (%v) — using in-memory rate limiter", err)
			_ = client.Close()
		} else {
			rdb = client
			log.Printf("Redis connected: %s (db %d)", cfg.Redis.Addr, cfg.Redis.DB)
		}
		cancel()
	}

	// stores
	userStore := store.NewUserStore(db)
	packageStore := store.NewPackageStore(db)
	domainStore := store.NewDomainStore(db)
	databaseStore := store.NewDatabaseStore(db)
	phpVersionStore := store.NewPHPVersionStore(db)
	phpExtensionStore := store.NewPHPExtensionStore(db)
	cronJobStore := store.NewCronJobStore(db)
	apiKeyStore := store.NewAPIKeyStore(db)
	auditLogStore := store.NewAuditLogStore(db)
	backupStore := store.NewBackupStore(db)
	subdomainStore := store.NewSubdomainStore(db)

	// handlers
	authH := handlers.NewAuthHandler(userStore, cfg.JWT.Secret, cfg.JWT.Expiry, cfg.JWT.TOTPKey)
	usersH := handlers.NewUserHandler(userStore, packageStore, domainStore, subdomainStore, databaseStore, cfg.Agent.Socket)
	packagesH := handlers.NewPackageHandler(packageStore)
	domainsH := handlers.NewDomainHandler(domainStore, subdomainStore, userStore, cfg.Agent.Socket, cfg.Paths.HomeBase)
	subdomainsH := handlers.NewSubdomainHandler(subdomainStore, domainStore, userStore, phpVersionStore, cfg.Agent.Socket, cfg.Paths.HomeBase)
	databasesH := handlers.NewDatabaseHandler(databaseStore, cfg.Agent.Socket, rdb)
	phpVersionsH := handlers.NewPHPVersionHandler(phpVersionStore)
	phpExtensionsH := handlers.NewPHPExtensionHandler(phpExtensionStore, userStore, cfg.Agent.Socket)
	cronJobsH := handlers.NewCronJobHandler(cronJobStore, userStore, packageStore, cfg.Agent.Socket)
	logsH := handlers.NewLogHandler(domainStore, userStore, cfg.Agent.Socket)
	installerH := handlers.NewInstallerHandler(domainStore, userStore, cfg.Agent.Socket)
	firewallH := handlers.NewFirewallHandler(cfg.Agent.Socket)
	antivirusH := handlers.NewAntivirusHandler(userStore, store.NewAntivirusAlertStore(db), cfg.Agent.Socket)
	backupTargetStore := store.NewBackupTargetStore(db)
	backupTargetsH := handlers.NewBackupTargetHandler(
		backupTargetStore,
		cfg.Agent.Socket,
		cfg.JWT.Secret,
		cfg.JWT.TOTPKey,
	)
	apiKeysH := handlers.NewAPIKeyHandler(apiKeyStore)
	auditLogsH := handlers.NewAuditLogHandler(auditLogStore)
	sslH := handlers.NewSSLHandler(domainStore, subdomainStore, cfg.Agent.Socket, cfg.LetsEncrypt.Email, cfg.LetsEncrypt.Staging)
	backupsH := handlers.NewBackupHandler(backupStore, userStore, databaseStore, cfg.Paths.HomeBase, cfg.Paths.BackupBase, cfg.Agent.Socket)
	backupsH.BackupTargets = backupTargetStore
	filesH := handlers.NewFileManagerHandler(userStore, cfg.Agent.Socket)
	systemH := handlers.NewSystemHandler(userStore, domainStore, databaseStore, cfg.Agent.Socket, version)
	terminalH := handlers.NewTerminalHandler(userStore, cfg.Agent.Socket)

	// router
	router := api.NewRouter(
		authH, usersH, packagesH, domainsH, subdomainsH, databasesH,
		phpVersionsH, phpExtensionsH, cronJobsH, logsH, installerH, firewallH, antivirusH, backupTargetsH, apiKeysH, auditLogsH, sslH, backupsH, filesH, systemH, terminalH,
		apiKeyStore, auditLogStore,
		rdb,
		cfg.JWT.Secret,
		cfg.Paths.FrontendDir,
	)
	engine := router.Setup()

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("ZensPanel API starting on %s", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
