package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/zenspanel/zenspanel/agent"
	agentbackup "github.com/zenspanel/zenspanel/agent/backup"
	agentcgroups "github.com/zenspanel/zenspanel/agent/cgroups"
	agentfilemanager "github.com/zenspanel/zenspanel/agent/filemanager"
	agentmysql "github.com/zenspanel/zenspanel/agent/mysql"
	agentnginx "github.com/zenspanel/zenspanel/agent/nginx"
	agentphpfpm "github.com/zenspanel/zenspanel/agent/phpfpm"
	agentssl "github.com/zenspanel/zenspanel/agent/ssl"
	agentterminal "github.com/zenspanel/zenspanel/agent/terminal"
	agentupdater "github.com/zenspanel/zenspanel/agent/updater"
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

	srv.Register("phpfpm.reload", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			PHPVersion string `json:"php_version"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentphpfpm.ReloadFPM(p.PHPVersion)
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

	srv.Register("cgroups.read_metrics", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username string `json:"username"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		ram, disk, cpu, err := agentcgroups.ReadMetrics(p.Username, cfg.Paths.HomeBase)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{
			"ram_used":  ram,
			"disk_used": disk,
			"cpu_pct":   cpu,
		}, nil
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
	// The agent needs admin-level access to CREATE DATABASE / CREATE USER /
	// GRANT — the panel-user DSN only has rights on its own schema, so we
	// require a separate admin DSN. Fall back to the panel DSN only if the
	// admin one isn't configured (which will fail loudly the first time a
	// db is provisioned, surfacing the misconfiguration).
	mysqlDSN := cfg.Agent.MySQLAdminDSN
	if mysqlDSN == "" {
		mysqlDSN = cfg.Database.DSN
		log.Printf("WARN: agent.mysql_admin_dsn not set; falling back to database.dsn — CREATE DATABASE will fail unless that user has admin grants")
	}
	mysqlClient, err := agentmysql.New(mysqlDSN)
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

	srv.Register("mysql.reset_password", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			DBUser      string `json:"db_user"`
			NewPassword string `json:"new_password"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, mysqlClient.ResetUserPassword(p.DBUser, p.NewPassword)
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
		chosen, err := agentuser.Create(p.Username, p.UID, cfg.Paths.HomeBase)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"uid": chosen}, nil
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

	// updater — self-update of the panel itself, all root-only ops
	// (git pull, go build, systemctl restart) routed through the agent
	// because the API runs as a non-root user.
	srv.Register("update.check", func(params json.RawMessage) (interface{}, error) {
		return agentupdater.Check(cfg.Paths.SrcDir)
	})
	srv.Register("update.run", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			DownloadURL string `json:"download_url"`
		}
		_ = json.Unmarshal(params, &p)
		err := agentupdater.Run(cfg.Paths.SrcDir, cfg.Paths.BinDir, cfg.Paths.FrontendDir, p.DownloadURL)
		return map[string]interface{}{"started": err == nil, "error": errMsg(err)}, nil
	})
	srv.Register("update.status", func(params json.RawMessage) (interface{}, error) {
		return agentupdater.Status(), nil
	})

	// backup restore
	srv.Register("backup.restore_files", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username    string `json:"username"`
			ArchivePath string `json:"archive_path"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentbackup.RestoreFiles(p.Username, cfg.Paths.HomeBase, cfg.Paths.BackupBase, p.ArchivePath)
	})

	srv.Register("backup.restore_db", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			DBName      string `json:"db_name"`
			ArchivePath string `json:"archive_path"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		dsn := cfg.Agent.MySQLAdminDSN
		if dsn == "" {
			dsn = cfg.Database.DSN
		}
		return nil, agentbackup.RestoreDB(p.DBName, p.ArchivePath, dsn, cfg.Paths.BackupBase)
	})

	// filemanager
	srv.Register("filemanager.list", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username string `json:"username"`
			Path     string `json:"path"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		entries, err := agentfilemanager.List(p.Username, cfg.Paths.HomeBase, p.Path)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"entries": entries}, nil
	})

	srv.Register("filemanager.read", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username string `json:"username"`
			Path     string `json:"path"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		content, err := agentfilemanager.Read(p.Username, cfg.Paths.HomeBase, p.Path)
		if err != nil {
			return nil, err
		}
		return map[string]interface{}{"content": content}, nil
	})

	srv.Register("filemanager.write", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username string `json:"username"`
			Path     string `json:"path"`
			Content  string `json:"content"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentfilemanager.Write(p.Username, cfg.Paths.HomeBase, p.Path, p.Content)
	})

	// filemanager.upload accepts a base64-encoded blob from the API
	// because the agent's JSON socket can't carry raw binary safely.
	// The API's Upload handler reads multipart/form-data, encodes, and
	// sends it through here. The agent decodes and writes — same home
	// jail as Write, just with []byte instead of string.
	srv.Register("filemanager.upload", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username string `json:"username"`
			Path     string `json:"path"`
			DataB64  string `json:"data_b64"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		data, err := base64.StdEncoding.DecodeString(p.DataB64)
		if err != nil {
			return nil, fmt.Errorf("decode base64: %w", err)
		}
		return nil, agentfilemanager.Upload(p.Username, cfg.Paths.HomeBase, p.Path, data)
	})

	srv.Register("filemanager.mkdir", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username string `json:"username"`
			Path     string `json:"path"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentfilemanager.Mkdir(p.Username, cfg.Paths.HomeBase, p.Path)
	})

	srv.Register("filemanager.rename", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username string `json:"username"`
			OldPath  string `json:"old_path"`
			NewPath  string `json:"new_path"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentfilemanager.Rename(p.Username, cfg.Paths.HomeBase, p.OldPath, p.NewPath)
	})

	srv.Register("filemanager.delete", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username string `json:"username"`
			Path     string `json:"path"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentfilemanager.Delete(p.Username, cfg.Paths.HomeBase, p.Path)
	})

	srv.Register("filemanager.chmod", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username string `json:"username"`
			Path     string `json:"path"`
			Mode     uint32 `json:"mode"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentfilemanager.Chmod(p.Username, cfg.Paths.HomeBase, p.Path, os.FileMode(p.Mode))
	})

	srv.Register("filemanager.copy", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username string `json:"username"`
			Src      string `json:"src"`
			Dst      string `json:"dst"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentfilemanager.Copy(p.Username, cfg.Paths.HomeBase, p.Src, p.Dst)
	})

	srv.Register("filemanager.compress", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username string `json:"username"`
			Src      string `json:"src"`
			Dst      string `json:"dst"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentfilemanager.Compress(p.Username, cfg.Paths.HomeBase, p.Src, p.Dst)
	})

	srv.Register("filemanager.extract", func(params json.RawMessage) (interface{}, error) {
		var p struct {
			Username string `json:"username"`
			Archive  string `json:"archive"`
			DstDir   string `json:"dst_dir"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, err
		}
		return nil, agentfilemanager.Extract(p.Username, cfg.Paths.HomeBase, p.Archive, p.DstDir)
	})

	log.Printf("ZensPanel Agent starting, socket: %s", cfg.Agent.Socket)
	if err := srv.Listen(); err != nil {
		log.Fatalf("agent listen: %v", err)
	}
}

// errMsg returns err.Error() or "" — used for RPC return shapes that
// surface a string error field even on success.
func errMsg(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
