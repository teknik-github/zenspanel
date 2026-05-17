package nginx

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"text/template"
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
	return ReloadNginx()
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
