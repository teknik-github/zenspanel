package ssl

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func certDir(sslBase, domain string) string {
	return filepath.Join(sslBase, domain)
}

func IssueLetsEncrypt(domain, email string, staging bool) error {
	args := []string{
		"--nginx",
		"-d", domain,
		"--email", email,
		"--agree-tos",
		"--non-interactive",
	}
	if staging {
		args = append(args, "--staging")
	}
	cmd := exec.Command("certbot", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("certbot failed: %w: %s", err, out)
	}
	return nil
}

func WriteCustomCert(sslBase, domain, certPEM, keyPEM string) error {
	if !strings.Contains(certPEM, "BEGIN CERTIFICATE") {
		return fmt.Errorf("invalid certificate PEM")
	}
	if !strings.Contains(keyPEM, "BEGIN") {
		return fmt.Errorf("invalid key PEM")
	}
	dir := certDir(sslBase, domain)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("mkdir ssl dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "cert.pem"), []byte(certPEM), 0600); err != nil {
		return fmt.Errorf("write cert: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "key.pem"), []byte(keyPEM), 0600); err != nil {
		return fmt.Errorf("write key: %w", err)
	}
	return nil
}

func RemoveCert(sslBase, domain string) error {
	return os.RemoveAll(certDir(sslBase, domain))
}
