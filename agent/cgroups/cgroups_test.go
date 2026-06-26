package cgroups

import (
	"os"
	"path/filepath"
	"testing"
)

func TestV71_ProcessRSSFallbackSumsUserProcesses(t *testing.T) {
	tmp := t.TempDir()
	oldProcRoot := procRoot
	oldProcessRSSPage := processRSSPage
	oldLookupUserUID := lookupUserUID
	oldProcessRSSCache := processRSSCache
	t.Cleanup(func() {
		procRoot = oldProcRoot
		processRSSPage = oldProcessRSSPage
		lookupUserUID = oldLookupUserUID
		processRSSCache = oldProcessRSSCache
	})

	procRoot = tmp
	processRSSPage = 4096
	processRSSCache = processRSSSnapshot{}
	lookupUserUID = func(username string) (uint32, error) {
		if username != "alice" {
			t.Fatalf("unexpected username %q", username)
		}
		return 1001, nil
	}

	writeProc(t, tmp, "101", "Uid:\t1001\t1001\t1001\t1001\n", "20 3 0 0 0 0 0\n")
	writeProc(t, tmp, "202", "Uid:\t0\t1001\t0\t0\n", "20 2 0 0 0 0 0\n")
	writeProc(t, tmp, "303", "Uid:\t1002\t1002\t1002\t1002\n", "20 9 0 0 0 0 0\n")
	if err := os.Mkdir(filepath.Join(tmp, "self"), 0755); err != nil {
		t.Fatal(err)
	}

	got, err := readProcessRSS("alice")
	if err != nil {
		t.Fatal(err)
	}
	want := int64(5 * 4096)
	if got != want {
		t.Fatalf("readProcessRSS() = %d, want %d", got, want)
	}
}

func writeProc(t *testing.T, root, pid, status, statm string) {
	t.Helper()
	dir := filepath.Join(root, pid)
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "status"), []byte(status), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "statm"), []byte(statm), 0644); err != nil {
		t.Fatal(err)
	}
}
