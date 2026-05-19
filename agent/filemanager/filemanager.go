package filemanager

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
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

// Chmod sets the permission bits on a file or directory. Mode is the
// numeric mode the operator typed in the UI — e.g. 0755 for rwxr-xr-x.
// We mask off the high bits so a careless caller can't set setuid/setgid
// on a user's file (those need root anyway, but masking makes intent
// explicit).
func Chmod(username, homeBase, rel string, mode os.FileMode) error {
	target, err := resolve(username, homeBase, rel)
	if err != nil {
		return err
	}
	return os.Chmod(target, mode&0777)
}

// Copy duplicates a file or directory tree to a new path. Both endpoints
// are validated through the same home-jail resolver, so the operator
// can't copy in or out of the jail. Existing destinations get overwritten
// without a confirm step — that matches how `cp` works at the shell.
func Copy(username, homeBase, srcRel, dstRel string) error {
	src, err := resolve(username, homeBase, srcRel)
	if err != nil {
		return err
	}
	dst, err := resolve(username, homeBase, dstRel)
	if err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat src: %w", err)
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst, info.Mode())
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("mkdir parent: %w", err)
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm())
	if err != nil {
		return fmt.Errorf("open dst: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy: %w", err)
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			info, _ := d.Info()
			mode := os.FileMode(0755)
			if info != nil {
				mode = info.Mode().Perm()
			}
			return os.MkdirAll(target, mode)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode())
	})
}

// Compress builds a .zip or .tar.gz from src and writes it to dst. The
// archive type is picked from dst's extension so the operator can choose
// in the UI by typing the filename. Anything else is rejected — we
// prefer to fail loudly than silently produce an unreadable archive.
func Compress(username, homeBase, srcRel, dstRel string) error {
	src, err := resolve(username, homeBase, srcRel)
	if err != nil {
		return err
	}
	dst, err := resolve(username, homeBase, dstRel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("mkdir parent: %w", err)
	}
	switch {
	case strings.HasSuffix(strings.ToLower(dst), ".zip"):
		return writeZip(src, dst)
	case strings.HasSuffix(strings.ToLower(dst), ".tar.gz"),
		strings.HasSuffix(strings.ToLower(dst), ".tgz"):
		return writeTarGz(src, dst)
	default:
		return fmt.Errorf("compress: dst must end in .zip, .tar.gz, or .tgz")
	}
}

func writeZip(src, dst string) error {
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create zip: %w", err)
	}
	defer out.Close()
	zw := zip.NewWriter(out)
	defer zw.Close()
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat src: %w", err)
	}
	base := filepath.Base(src)
	if !srcInfo.IsDir() {
		// Single file — just add it under its basename.
		return addFileToZip(zw, src, base)
	}
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		nameInZip := filepath.ToSlash(filepath.Join(base, rel))
		if d.IsDir() {
			if rel == "." {
				return nil
			}
			_, err := zw.Create(nameInZip + "/")
			return err
		}
		return addFileToZip(zw, path, nameInZip)
	})
}

func addFileToZip(zw *zip.Writer, path, nameInZip string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	hdr, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	hdr.Name = nameInZip
	hdr.Method = zip.Deflate
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	_, err = io.Copy(w, in)
	return err
}

func writeTarGz(src, dst string) error {
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create tar.gz: %w", err)
	}
	defer out.Close()
	gz := gzip.NewWriter(out)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	base := filepath.Base(src)
	if !srcInfo.IsDir() {
		return addFileToTar(tw, src, base, srcInfo)
	}
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		nameInTar := filepath.ToSlash(filepath.Join(base, rel))
		return addFileToTar(tw, path, nameInTar, info)
	})
}

func addFileToTar(tw *tar.Writer, path, nameInTar string, info os.FileInfo) error {
	hdr, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	hdr.Name = nameInTar
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	_, err = io.Copy(tw, in)
	return err
}

// Extract reads a .zip or .tar.gz/.tgz archive and writes its contents
// under dstDir. Every entry's destination is run back through resolve()
// so a malicious archive can't escape the home jail with ../ paths or
// absolute names.
func Extract(username, homeBase, archiveRel, dstDirRel string) error {
	archive, err := resolve(username, homeBase, archiveRel)
	if err != nil {
		return err
	}
	dstDir, err := resolve(username, homeBase, dstDirRel)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return fmt.Errorf("mkdir dst: %w", err)
	}
	lower := strings.ToLower(archive)
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return extractZip(archive, dstDir, username, homeBase, dstDirRel)
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return extractTarGz(archive, dstDir, username, homeBase, dstDirRel)
	default:
		return fmt.Errorf("extract: archive must end in .zip, .tar.gz, or .tgz")
	}
}

func extractZip(archive, dstDir, username, homeBase, dstDirRel string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()
	for _, f := range zr.File {
		// Re-validate every entry through resolve so a path with ../
		// in it can't write outside the user's home.
		entryRel := filepath.Join(dstDirRel, f.Name)
		target, err := resolve(username, homeBase, entryRel)
		if err != nil {
			return fmt.Errorf("entry %q: %w", f.Name, err)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, f.Mode()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, f.Mode())
		if err != nil {
			return fmt.Errorf("create %q: %w", f.Name, err)
		}
		in, err := f.Open()
		if err != nil {
			out.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		in.Close()
		out.Close()
		if copyErr != nil {
			return fmt.Errorf("write %q: %w", f.Name, copyErr)
		}
	}
	return nil
}

func extractTarGz(archive, dstDir, username, homeBase, dstDirRel string) error {
	in, err := os.Open(archive)
	if err != nil {
		return fmt.Errorf("open tar: %w", err)
	}
	defer in.Close()
	gz, err := gzip.NewReader(in)
	if err != nil {
		return fmt.Errorf("gunzip: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}
		entryRel := filepath.Join(dstDirRel, hdr.Name)
		target, err := resolve(username, homeBase, entryRel)
		if err != nil {
			return fmt.Errorf("entry %q: %w", hdr.Name, err)
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode))
			if err != nil {
				return fmt.Errorf("create %q: %w", hdr.Name, err)
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return fmt.Errorf("write %q: %w", hdr.Name, err)
			}
			out.Close()
		}
		// Symlinks, hardlinks, devices, etc. are intentionally skipped:
		// a panel user's archive shouldn't need them, and supporting
		// them widens the security surface a lot.
	}
}
