package main

import (
	"fmt"
	"log"

	"github.com/zenspanel/zenspanel/internal/api"
	"github.com/zenspanel/zenspanel/internal/api/handlers"
	"github.com/zenspanel/zenspanel/internal/config"
	"github.com/zenspanel/zenspanel/internal/store"
)

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

	// stores
	userStore := store.NewUserStore(db)
	packageStore := store.NewPackageStore(db)
	domainStore := store.NewDomainStore(db)
	databaseStore := store.NewDatabaseStore(db)
	phpVersionStore := store.NewPHPVersionStore(db)
	apiKeyStore := store.NewAPIKeyStore(db)
	auditLogStore := store.NewAuditLogStore(db)
	backupStore := store.NewBackupStore(db)

	// handlers
	authH := handlers.NewAuthHandler(userStore, cfg.JWT.Secret, cfg.JWT.Expiry)
	usersH := handlers.NewUserHandler(userStore, packageStore, cfg.Agent.Socket)
	packagesH := handlers.NewPackageHandler(packageStore)
	domainsH := handlers.NewDomainHandler(domainStore, userStore, cfg.Agent.Socket, cfg.Paths.HomeBase)
	databasesH := handlers.NewDatabaseHandler(databaseStore, cfg.Agent.Socket)
	phpVersionsH := handlers.NewPHPVersionHandler(phpVersionStore)
	apiKeysH := handlers.NewAPIKeyHandler(apiKeyStore)
	auditLogsH := handlers.NewAuditLogHandler(auditLogStore)
	sslH := handlers.NewSSLHandler(domainStore, cfg.Agent.Socket, cfg.LetsEncrypt.Email, cfg.LetsEncrypt.Staging)
	backupsH := handlers.NewBackupHandler(backupStore, userStore, databaseStore, cfg.Paths.HomeBase, cfg.Paths.BackupBase)
	filesH := handlers.NewFileManagerHandler(userStore, cfg.Agent.Socket)

	// router
	router := api.NewRouter(
		authH, usersH, packagesH, domainsH, databasesH,
		phpVersionsH, apiKeysH, auditLogsH, sslH, backupsH, filesH,
		apiKeyStore, auditLogStore,
		cfg.JWT.Secret,
	)
	engine := router.Setup()

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("ZensPanel API starting on %s", addr)
	if err := engine.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
