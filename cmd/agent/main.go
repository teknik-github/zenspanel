package main

import (
	"encoding/json"
	"log"

	"github.com/zenspanel/zenspanel/agent"
	agentcgroups "github.com/zenspanel/zenspanel/agent/cgroups"
	agentmysql "github.com/zenspanel/zenspanel/agent/mysql"
	agentnginx "github.com/zenspanel/zenspanel/agent/nginx"
	agentphpfpm "github.com/zenspanel/zenspanel/agent/phpfpm"
	agentssl "github.com/zenspanel/zenspanel/agent/ssl"
	agentterminal "github.com/zenspanel/zenspanel/agent/terminal"
	agentuser "github.com/zenspanel/zenspanel/agent/user"
	"github.com/zenspanel/zenspanel/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	srv := agent.NewServer(cfg.Agent.Socket)
	srv.SetSocketGroup(cfg.Agent.SocketGroup)

	// nginx
	srv.Register("nginx.create_vhost", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Domain     string `json:"domain"`
			Username   string `json:"username"`
			PHPVersion string `json:"php_version"`
			DocRoot    string `json:"doc_root"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentnginx.CreateVhost(cfg.Paths.NginxConf, p.Domain, p.Username, p.PHPVersion, p.DocRoot)
	})

	srv.Register("nginx.delete_vhost", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Domain string `json:"domain"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentnginx.DeleteVhost(cfg.Paths.NginxConf, p.Domain)
	})

	srv.Register("nginx.suspend_vhost", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Domain string `json:"domain"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentnginx.SuspendVhost(cfg.Paths.NginxConf, p.Domain)
	})

	srv.Register("nginx.reload", func(params json.RawMessage) (interface{}, error) {
		return nil, agentnginx.ReloadNginx()
	})

	// phpfpm
	srv.Register("phpfpm.create_pool", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username   string `json:"username"`
			PHPVersion string `json:"php_version"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentphpfpm.CreatePool(cfg.Paths.PHPPoolBase, p.Username, p.PHPVersion)
	})

	srv.Register("phpfpm.delete_pool", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username   string `json:"username"`
			PHPVersion string `json:"php_version"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentphpfpm.DeletePool(cfg.Paths.PHPPoolBase, p.Username, p.PHPVersion)
	})

	// cgroups
	srv.Register("cgroups.create_slice", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username    string `json:"username"`
			CPUQuota    int    `json:"cpu_quota"`
			MemoryLimit int64  `json:"memory_limit"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentcgroups.CreateSlice(p.Username, p.CPUQuota, p.MemoryLimit)
	})

	srv.Register("cgroups.update_slice", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username    string `json:"username"`
			CPUQuota    int    `json:"cpu_quota"`
			MemoryLimit int64  `json:"memory_limit"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentcgroups.UpdateSlice(p.Username, p.CPUQuota, p.MemoryLimit)
	})

	srv.Register("cgroups.delete_slice", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username string `json:"username"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentcgroups.DeleteSlice(p.Username)
	})

	// ssl
	srv.Register("ssl.issue_letsencrypt", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Domain  string `json:"domain"`
			Email   string `json:"email"`
			Staging bool   `json:"staging"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentssl.IssueLetsEncrypt(p.Domain, p.Email, p.Staging)
	})

	srv.Register("ssl.write_custom_cert", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Domain  string `json:"domain"`
			CertPEM string `json:"cert_pem"`
			KeyPEM  string `json:"key_pem"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentssl.WriteCustomCert(cfg.Paths.SSLBase, p.Domain, p.CertPEM, p.KeyPEM)
	})

	srv.Register("ssl.remove_cert", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Domain string `json:"domain"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentssl.RemoveCert(cfg.Paths.SSLBase, p.Domain)
	})

	// mysql
	mysqlClient, err := agentmysql.New(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("failed to connect agent mysql: %v", err)
	}

	srv.Register("mysql.create_database", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			DBName     string `json:"db_name"`
			DBUser     string `json:"db_user"`
			DBPassword string `json:"db_password"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, mysqlClient.CreateDatabase(p.DBName, p.DBUser, p.DBPassword)
	})

	srv.Register("mysql.drop_database", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			DBName string `json:"db_name"`
			DBUser string `json:"db_user"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, mysqlClient.DropDatabase(p.DBName, p.DBUser)
	})

	// terminal
	srv.Register("terminal.spawn", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username string `json:"username"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		session, err := agentterminal.SpawnSession(p.Username, cfg.Paths.HomeBase)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"pid": session.Cmd.Process.Pid}, nil
	})

	// user
	srv.Register("user.create", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username string `json:"username"`
			UID      int    `json:"uid"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentuser.Create(p.Username, p.UID, cfg.Paths.HomeBase)
	})

	srv.Register("user.delete", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username string `json:"username"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentuser.Delete(p.Username)
	})

	log.Printf("ZensPanel Agent starting, socket: %s", cfg.Agent.Socket)
	if err := srv.Listen(); err != nil {
		log.Fatalf("agent listen: %v", err)
	}
}
