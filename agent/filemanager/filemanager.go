package filemanager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zenspanel/zenspanel/agent/safe"
)

// maxReadSize caps the size of a file we'll Read into memory and ship over
// the agent socket. The Monaco editor on the User Panel is fine with up to
// a few MB; larger files should be edited via terminal/SFTP.
const maxReadSize int64 = 4 * 1024 * 1024 // 4 MiB

// maxUploadSize caps inbound binary uploads. The agent JSON socket isn't
// great for very large blobs (the API has to base64-encode the bytes
// before forwarding, which inflates them ~33%), and the upload handler
// holds the whole file in memory at once. 64 MiB matches Gin's
// MaxMultipartMemory default and is more than enough for typical web
// hosting assets — anything larger should travel over SFTP.
const maxUploadSize int64 = 64 * 1024 * 1024 // 64 MiB

// Entry is one row of a directory listing.
type Entry struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"is_dir"`
	ModTime int64  `json:"mod_time"` // unix seconds
	Mode    string `json:"mode"`     // -rw-r--r-- style
}

// resolve takes a user-supplied relative path and returns the absolute
// filesystem path, refusing anything that escapes the user's home jail.
// We use filepath.Clean + Abs + EvalSymlinks so that:
//   - "../etc/passwd" is rejected (Clean collapses but Abs reveals escape)
//   - a symlink inside the home pointing at /etc is rejected (EvalSymlinks)
//
// This is the single security gate every public function in this package
// goes through. Bypassing it would give a logged-in user root file access.
func resolve(username, homeBase, rel string) (string, error) {
	if err := safe.Username(username); err != nil {
		return "", err
	}
	homeDir, err := filepath.Abs(filepath.Join(homeBase, username))
	if err != nil {
		return "", fmt.Errorf("abs home: %w", err)
	}
	// Normalise the requested path so leading "/" or "./" or "" all map to
	// the user's home root.
	rel = strings.TrimPrefix(filepath.Clean("/"+rel), "/")
	target := filepath.Clean(filepath.Join(homeDir, rel))
	if target != homeDir && !strings.HasPrefix(target, homeDir+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes home directory")
	}
	// Resolve symlinks if the target exists, so a symlink inside the home
	// can't be a tunnel out. Non-existent targets are fine — we may be
	// about to create them.
	if info, err := os.Lstat(target); err == nil && info.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(target)
		if err != nil {
			return "", fmt.Errorf("eval symlink: %w", err)
		}
		if resolved != homeDir && !strings.HasPrefix(resolved, homeDir+string(filepath.Separator)) {
			return "", fmt.Errorf("symlink escapes home directory")
		}
		target = resolved
	}
	return target, nil
}

func List(username, homeBase, rel string) ([]Entry, error) {
	target, err := resolve(username, homeBase, rel)
	if err != nil {
		return nil, err
	}
	dir, err := os.Open(target)
	if err != nil {
		return nil, fmt.Errorf("open dir: %w", err)
	}
	defer dir.Close()
	infos, err := dir.Readdir(-1)
	if err != nil {
		return nil, fmt.Errorf("readdir: %w", err)
	}
	out := make([]Entry, 0, len(infos))
	for _, info := range infos {
		out = append(out, Entry{
			Name:    info.Name(),
			Size:    info.Size(),
			IsDir:   info.IsDir(),
			ModTime: info.ModTime().Unix(),
			Mode:    info.Mode().String(),
		})
	}
	return out, nil
}

func Read(username, homeBase, rel string) (string, error) {
	target, err := resolve(username, homeBase, rel)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(target)
	if err != nil {
		return "", fmt.Errorf("stat: %w", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("path is a directory")
	}
	if info.Size() > maxReadSize {
		return "", fmt.Errorf("file too large (max %d bytes)", maxReadSize)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return "", fmt.Errorf("read: %w", err)
	}
	return string(data), nil
}

func Write(username, homeBase, rel, content string) error {
	target, err := resolve(username, homeBase, rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return fmt.Errorf("mkdir parent: %w", err)
	}
	return os.WriteFile(target, []byte(content), 0644)
}

// Upload writes a binary blob to the user's home jail. Same security
// gate as Write (path resolves through resolve()), but accepts raw
// []byte so the caller can ship binaries (images, archives, etc.) that
// would corrupt over the JSON-string Write path. The byte slice has
// already been base64-decoded in cmd/agent/main.go before reaching here;
// the size cap is defensive in case the API forwards something larger
// than its own limit.
func Upload(username, homeBase, rel string, data []byte) error {
	if int64(len(data)) > maxUploadSize {
		return fmt.Errorf("file too large (max %d bytes)", maxUploadSize)
	}
	target, err := resolve(username, homeBase, rel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return fmt.Errorf("mkdir parent: %w", err)
	}
	return os.WriteFile(target, data, 0644)
}

func Mkdir(username, homeBase, rel string) error {
	target, err := resolve(username, homeBase, rel)
	if err != nil {
		return err
	}
	return os.MkdirAll(target, 0755)
}

func Rename(username, homeBase, oldRel, newRel string) error {
	oldPath, err := resolve(username, homeBase, oldRel)
	if err != nil {
		return err
	}
	newPath, err := resolve(username, homeBase, newRel)
	if err != nil {
		return err
	}
	return os.Rename(oldPath, newPath)
}

func Delete(username, homeBase, rel string) error {
	target, err := resolve(username, homeBase, rel)
	if err != nil {
		return err
	}
	// Refuse to nuke the user's home root from this endpoint — that's a
	// foot-gun even on purpose, and there's no undo button.
	homeDir, _ := filepath.Abs(filepath.Join(homeBase, username))
	if target == homeDir {
		return fmt.Errorf("refusing to delete home root")
	}
	return os.RemoveAll(target)
}
