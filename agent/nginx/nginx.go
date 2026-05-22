package nginx

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	osuser "os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/zenspanel/zenspanel/agent/safe"
)

var validDomain = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-\.]{0,253}[a-zA-Z0-9]$`)

const vhostTmpl = `server {
    listen 80;
    server_name {{.Domain}};
    root {{.DocRoot}};
    index index.php index.html;

    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location ~ \.php$ {
        fastcgi_pass unix:{{.FPMSocket}};
        fastcgi_index index.php;
        include fastcgi_params;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
    }
}
`

const suspendTmpl = `server {
    listen 80;
    server_name {{.Domain}};
    return 503;
}
`

type vhostData struct {
	Domain    string
	DocRoot   string
	FPMSocket string
}

func confPath(nginxConf, domain string) string {
	return filepath.Join(nginxConf, domain+".conf")
}

func CreateVhost(nginxConf, domain, username, phpVersion, docRoot string) error {
	if !validDomain.MatchString(domain) {
		return fmt.Errorf("invalid domain: %s", domain)
	}
	tmpl := template.Must(template.New("vhost").Parse(vhostTmpl))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vhostData{
		Domain:    domain,
		DocRoot:   docRoot,
		FPMSocket: fmt.Sprintf("/run/php/zenspanel-%s-%s.sock", username, phpVersion),
	}); err != nil {
		return fmt.Errorf("template: %w", err)
	}
	if err := os.MkdirAll(nginxConf, 0755); err != nil {
		return fmt.Errorf("mkdir nginx conf: %w", err)
	}
	if err := os.WriteFile(confPath(nginxConf, domain), buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write vhost: %w", err)
	}
	if err := ensureDocRoot(docRoot, domain, username); err != nil {
		return fmt.Errorf("ensure docroot: %w", err)
	}
	return ReloadNginx()
}

// ensureDocRoot creates the website's document root if it doesn't exist,
// chowns it to the panel Linux user, and seeds an index.html so the very
// first hit serves a friendly "coming soon" page instead of an nginx
// default 404. Re-running on an existing vhost is safe — we only write
// the placeholder when the directory has no index file.
//
// Permissions are set explicitly via os.Chmod after every create
// because os.MkdirAll / os.WriteFile pass the mode through the process
// umask: agent inherits umask from systemd (often 0027 on Ubuntu),
// which silently strips the read bit for "others" — turning 0755 into
// 0750 and 0644 into 0640. Nginx as www-data then can't traverse or
// read the file, surfacing as "Access denied" / "File not found.".
func ensureDocRoot(docRoot, domain, username string) error {
	if err := os.MkdirAll(docRoot, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	// Override umask: every dir on the chain (docroot itself + parents
	// up to public_html) needs 0755 so others have read+traverse for
	// listing static assets via nginx autoindex if enabled.
	if err := os.Chmod(docRoot, 0755); err != nil {
		return fmt.Errorf("chmod docroot: %w", err)
	}
	uid, gid := -1, -1
	if u, err := osuser.Lookup(username); err == nil {
		if v, err := strconv.Atoi(u.Uid); err == nil {
			uid = v
		}
		if v, err := strconv.Atoi(u.Gid); err == nil {
			gid = v
		}
	}
	if uid >= 0 && gid >= 0 {
		// Walk up to the user's home and chown each parent we created
		// so the user can list and traverse them. We stop at home_base
		// to avoid touching /home itself.
		_ = chownTree(docRoot, uid, gid)
	}

	// Skip seeding if any index file already exists — operator may have
	// uploaded their site already, or this is a vhost recreate after
	// suspend/unsuspend and we don't want to clobber their files.
	for _, name := range []string{"index.html", "index.htm", "index.php"} {
		if _, err := os.Stat(filepath.Join(docRoot, name)); err == nil {
			return nil
		}
	}

	indexHTML := fmt.Sprintf(`<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>%s</title>
<style>
  body{font-family:system-ui,-apple-system,Segoe UI,Roboto,sans-serif;
    background:#f9fafb;color:#374151;display:flex;align-items:center;
    justify-content:center;height:100vh;margin:0}
  .box{text-align:center;padding:2rem}
  .domain{font-size:1.75rem;font-weight:700;color:#4f46e5;margin-bottom:.5rem}
  .sub{color:#9ca3af;font-size:.95rem;line-height:1.5}
  .hint{margin-top:1.5rem;font-size:.8rem;color:#d1d5db}
</style></head>
<body><div class="box">
<div class="domain">%s</div>
<div class="sub">Website coming soon.<br>
Upload your files via the panel File Manager to replace this page.</div>
<div class="hint">Powered by ZensPanel</div>
</div></body></html>
`, domain, domain)
	indexPath := filepath.Join(docRoot, "index.html")
	if err := os.WriteFile(indexPath, []byte(indexHTML), 0644); err != nil {
		return fmt.Errorf("write index.html: %w", err)
	}
	// Bypass umask — see comment at top of function.
	if err := os.Chmod(indexPath, 0644); err != nil {
		return fmt.Errorf("chmod index.html: %w", err)
	}
	if uid >= 0 && gid >= 0 {
		_ = os.Chown(indexPath, uid, gid)
	}
	return nil
}

// chownTree walks docRoot and every ancestor up to (but not including)
// the user's home base and chowns them to the user. We stop the walk
// once we hit /home or a path component the user shouldn't own.
func chownTree(docRoot string, uid, gid int) error {
	// Chown the docroot itself first.
	if err := os.Chown(docRoot, uid, gid); err != nil {
		return err
	}
	// Walk up parents until we hit /home or /. Each parent gets chowned
	// only if the user owns one level deeper — we don't want to chown
	// /home/zenspanel/<username>'s parent, only the public_html branch.
	for parent := filepath.Dir(docRoot); ; parent = filepath.Dir(parent) {
		if parent == "/" || parent == "/home" || parent == filepath.Dir(parent) {
			break
		}
		base := filepath.Base(parent)
		if base == "public_html" {
			_ = os.Chown(parent, uid, gid)
			continue
		}
		// Stop at the user's home dir level.
		break
	}
	return nil
}

func DeleteVhost(nginxConf, domain string) error {
	if err := os.Remove(confPath(nginxConf, domain)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove vhost: %w", err)
	}
	return ReloadNginx()
}

func SuspendVhost(nginxConf, domain string) error {
	tmpl := template.Must(template.New("suspend").Parse(suspendTmpl))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vhostData{Domain: domain}); err != nil {
		return fmt.Errorf("template: %w", err)
	}
	if err := os.WriteFile(confPath(nginxConf, domain), buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write suspend vhost: %w", err)
	}
	return ReloadNginx()
}

func ReloadNginx() error {
	test := exec.Command("nginx", "-t")
	if out, err := test.CombinedOutput(); err != nil {
		return fmt.Errorf("nginx -t failed: %w: %s", err, out)
	}
	reload := exec.Command("systemctl", "reload", "nginx")
	if out, err := reload.CombinedOutput(); err != nil {
		return fmt.Errorf("nginx reload failed: %w: %s", err, out)
	}
	return nil
}

// Redirect holds a single redirect rule for SyncRedirects.
type Redirect struct {
	SourcePath string `json:"source_path"`
	DestURL    string `json:"dest_url"`
	Type       string `json:"type"` // "301" or "302"
	Enabled    bool   `json:"enabled"`
}

// redirectSnippetPath returns the path for the per-domain redirect snippet.
func redirectSnippetPath(nginxConf, domain string) string {
	return filepath.Join(nginxConf, domain+".redirects.conf")
}

// SyncRedirects rewrites the redirect snippet for a domain and reloads nginx.
// Uses text/template — no shell string interpolation (V55).
// Only enabled redirects are written; disabled ones are omitted.
func SyncRedirects(nginxConf, domain string, redirects []Redirect) error {
	if err := safe.Domain(domain); err != nil {
		return err
	}

	const redirectTmpl = `# Managed by ZensPanel — do not edit manually
{{- range .}}
{{- if .Enabled}}
location = {{.SourcePath}} {
    return {{.Type}} {{.DestURL}};
}
{{- end}}
{{- end}}
`
	tmpl, err := template.New("redirects").Parse(redirectTmpl)
	if err != nil {
		return fmt.Errorf("parse redirect template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, redirects); err != nil {
		return fmt.Errorf("execute redirect template: %w", err)
	}

	snippetPath := redirectSnippetPath(nginxConf, domain)
	if len(redirects) == 0 || buf.Len() == 0 {
		// No redirects — remove snippet if it exists.
		_ = os.Remove(snippetPath)
	} else {
		if err := os.WriteFile(snippetPath, buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("write redirect snippet: %w", err)
		}
	}

	// Ensure the main vhost includes the snippet. If the vhost doesn't
	// exist yet, skip — redirects will be picked up when vhost is created.
	vhostFile := confPath(nginxConf, domain)
	if _, err := os.Stat(vhostFile); err == nil {
		content, err := os.ReadFile(vhostFile)
		if err == nil {
			includeDirective := fmt.Sprintf("include %s;", snippetPath)
			if !strings.Contains(string(content), includeDirective) {
				// Inject include before the closing brace of the server block.
				updated := strings.Replace(string(content), "\n}",
					fmt.Sprintf("\n    %s\n}", includeDirective), 1)
				_ = os.WriteFile(vhostFile, []byte(updated), 0644)
			}
		}
	}

	return ReloadNginx()
}


// SetHotlinkProtection writes or removes a hotlink protection block in the
// domain's vhost. Uses text/template — no shell string interpolation (V55).
// Only applies to static asset extensions so API/WS paths are unaffected (V54).
func SetHotlinkProtection(nginxConf, domain string, enabled bool, allowedDomains []string) error {
	if err := safe.Domain(domain); err != nil {
		return err
	}

	snippetPath := filepath.Join(nginxConf, domain+".hotlink.conf")

	if !enabled {
		_ = os.Remove(snippetPath)
		vhostFile := confPath(nginxConf, domain)
		if content, err := os.ReadFile(vhostFile); err == nil {
			includeDirective := fmt.Sprintf("include %s;", snippetPath)
			updated := strings.ReplaceAll(string(content), "\n    "+includeDirective, "")
			_ = os.WriteFile(vhostFile, []byte(updated), 0644)
		}
		return ReloadNginx()
	}

	referers := []string{"$server_name"}
	for _, d := range allowedDomains {
		if d != "" {
			referers = append(referers, d)
		}
	}

	const hotlinkTmpl = "# Managed by ZensPanel\nlocation ~* \\.(jpg|jpeg|png|gif|webp|svg|ico|woff|woff2|ttf|eot)$ {\n    valid_referers none blocked {{range .}}{{.}} {{end}};\n    if ($invalid_referer) {\n        return 403;\n    }\n}\n"

	tmpl, err := template.New("hotlink").Parse(hotlinkTmpl)
	if err != nil {
		return fmt.Errorf("parse hotlink template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, referers); err != nil {
		return fmt.Errorf("execute hotlink template: %w", err)
	}
	if err := os.WriteFile(snippetPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write hotlink snippet: %w", err)
	}

	vhostFile := confPath(nginxConf, domain)
	if content, err := os.ReadFile(vhostFile); err == nil {
		includeDirective := fmt.Sprintf("include %s;", snippetPath)
		if !strings.Contains(string(content), includeDirective) {
			updated := strings.Replace(string(content), "\n}", fmt.Sprintf("\n    %s\n}", includeDirective), 1)
			_ = os.WriteFile(vhostFile, []byte(updated), 0644)
		}
	}

	return ReloadNginx()
}

// SuspendAllVhosts replaces every vhost config owned by username with a
// 503 suspend page. Ownership is determined by the username embedded in
// the pool socket path inside the config file. Reloads nginx once after
// all files are rewritten so there is only one reload per suspend op.
func SuspendAllVhosts(nginxConf, username string) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	entries, err := os.ReadDir(nginxConf)
	if err != nil {
		return fmt.Errorf("readdir %s: %w", nginxConf, err)
	}
	changed := false
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".conf") {
			continue
		}
		path := filepath.Join(nginxConf, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if !strings.Contains(string(data), "zenspanel-"+username+"-") {
			continue
		}
		domain := strings.TrimSuffix(e.Name(), ".conf")
		if !strings.Contains(domain, ".") {
			continue
		}
		_ = os.WriteFile(path+".presuspend", data, 0644)
		tmpl := template.Must(template.New("s").Parse(suspendTmpl))
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, vhostData{Domain: domain}); err != nil {
			continue
		}
		_ = os.WriteFile(path, buf.Bytes(), 0644)
		changed = true
	}
	if changed {
		return ReloadNginx()
	}
	return nil
}

// UnsuspendAllVhosts restores vhost configs from .presuspend backups for
// all domains owned by username.
func UnsuspendAllVhosts(nginxConf, username string) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	entries, err := os.ReadDir(nginxConf)
	if err != nil {
		return fmt.Errorf("readdir %s: %w", nginxConf, err)
	}
	changed := false
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".conf.presuspend") {
			continue
		}
		backupPath := filepath.Join(nginxConf, e.Name())
		data, err := os.ReadFile(backupPath)
		if err != nil {
			continue
		}
		if !strings.Contains(string(data), "zenspanel-"+username+"-") {
			continue
		}
		origPath := strings.TrimSuffix(backupPath, ".presuspend")
		_ = os.WriteFile(origPath, data, 0644)
		_ = os.Remove(backupPath)
		changed = true
	}
	if changed {
		return ReloadNginx()
	}
	return nil
}
