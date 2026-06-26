package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateDownloadURLAllowsGitHubHTTPS(t *testing.T) {
	if err := validateDownloadURL("https://github.com/teknik-github/zenspanel/releases/download/v2.1.1/zenspanel-linux-amd64.tar.gz"); err != nil {
		t.Fatal(err)
	}
	if err := validateDownloadURL("https://objects.githubusercontent.com/github-production-release-asset/example"); err != nil {
		t.Fatal(err)
	}
}

func TestValidateDownloadURLRejectsUnsafeHosts(t *testing.T) {
	for _, raw := range []string{
		"http://github.com/teknik-github/zenspanel/archive.tar.gz",
		"https://example.com/zenspanel.tar.gz",
		"not a url",
	} {
		if err := validateDownloadURL(raw); err == nil {
			t.Fatalf("validateDownloadURL(%q) = nil, want error", raw)
		}
	}
}

func TestInstallExecutableAtomicReplacesTarget(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")
	if err := os.WriteFile(src, []byte("new"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := installExecutableAtomic(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("dst = %q, want new", got)
	}
}
