package handlers

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type FileManagerHandler struct {
	users     *store.UserStore
	agentSock string
}

func NewFileManagerHandler(users *store.UserStore, agentSock string) *FileManagerHandler {
	return &FileManagerHandler{users: users, agentSock: agentSock}
}

// caller resolves the caller's panel username so the handler can ask the
// agent to operate inside that user's home jail. Returns 401 if the
// session has no user (shouldn't happen behind JWTMiddleware) or 404 if
// the row vanished after login.
func (h *FileManagerHandler) caller(c *gin.Context) (string, bool) {
	uid := auth.GetUserID(c)
	if uid == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "no user in context"})
		return "", false
	}
	user, err := h.users.GetByID(uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return "", false
	}
	return user.Username, true
}

func (h *FileManagerHandler) List(c *gin.Context) {
	username, ok := h.caller(c)
	if !ok {
		return
	}
	var resp struct {
		Entries []map[string]interface{} `json:"entries"`
	}
	err := agent.NewClient(h.agentSock).Call("filemanager.list", map[string]interface{}{
		"username": username,
		"path":     c.Query("path"),
	}, &resp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"entries": resp.Entries})
}

func (h *FileManagerHandler) Read(c *gin.Context) {
	username, ok := h.caller(c)
	if !ok {
		return
	}
	var resp struct {
		Content string `json:"content"`
	}
	err := agent.NewClient(h.agentSock).Call("filemanager.read", map[string]interface{}{
		"username": username,
		"path":     c.Query("path"),
	}, &resp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"content": resp.Content})
}

func (h *FileManagerHandler) Write(c *gin.Context) {
	username, ok := h.caller(c)
	if !ok {
		return
	}
	var req struct {
		Path    string `json:"path" binding:"required"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := agent.NewClient(h.agentSock).Call("filemanager.write", map[string]interface{}{
		"username": username,
		"path":     req.Path,
		"content":  req.Content,
	}, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "saved"})
}

func (h *FileManagerHandler) Mkdir(c *gin.Context) {
	username, ok := h.caller(c)
	if !ok {
		return
	}
	var req struct {
		Path string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := agent.NewClient(h.agentSock).Call("filemanager.mkdir", map[string]interface{}{
		"username": username,
		"path":     req.Path,
	}, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "created"})
}

func (h *FileManagerHandler) Rename(c *gin.Context) {
	username, ok := h.caller(c)
	if !ok {
		return
	}
	var req struct {
		OldPath string `json:"old_path" binding:"required"`
		NewPath string `json:"new_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := agent.NewClient(h.agentSock).Call("filemanager.rename", map[string]interface{}{
		"username": username,
		"old_path": req.OldPath,
		"new_path": req.NewPath,
	}, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "renamed"})
}

func (h *FileManagerHandler) Delete(c *gin.Context) {
	username, ok := h.caller(c)
	if !ok {
		return
	}
	err := agent.NewClient(h.agentSock).Call("filemanager.delete", map[string]interface{}{
		"username": username,
		"path":     c.Query("path"),
	}, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// maxUploadSize must stay in sync with agent/filemanager.maxUploadSize.
// We enforce it here to refuse oversized uploads early — without this
// guard a 1 GB request would fully buffer in memory before the agent
// could reject it.
const maxUploadSize = 64 << 20 // 64 MiB

// Upload accepts a multipart/form-data POST with a "file" part and a
// "path" form value (the destination directory inside the user's home).
// We base64-encode the bytes before forwarding to the agent because the
// agent's JSON socket can't transport binary safely. The encoding inflates
// the in-memory copy ~33%; that's the cost of going through the agent's
// security jail instead of letting the API write directly to the user's
// home (which would bypass the path-resolution check).
func (h *FileManagerHandler) Upload(c *gin.Context) {
	username, ok := h.caller(c)
	if !ok {
		return
	}

	destDir := c.PostForm("path")
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing file: " + err.Error()})
		return
	}
	defer file.Close()

	if header.Size > maxUploadSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file exceeds 64 MiB limit"})
		return
	}

	// LimitReader caps memory even if Content-Length lied to us.
	data, err := io.ReadAll(io.LimitReader(file, maxUploadSize+1))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "read upload: " + err.Error()})
		return
	}
	if int64(len(data)) > maxUploadSize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "file exceeds 64 MiB limit"})
		return
	}

	uploadPath := path.Join(destDir, header.Filename)
	encoded := base64.StdEncoding.EncodeToString(data)

	err = agent.NewClient(h.agentSock).Call("filemanager.upload", map[string]interface{}{
		"username": username,
		"path":     uploadPath,
		"data_b64": encoded,
	}, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "uploaded", "path": uploadPath, "size": header.Size})
}

// Chmod changes permission bits on a file or directory. Mode comes in
// as either an octal string ("0755") or a plain decimal — we normalize
// here so the frontend can send whichever is convenient.
func (h *FileManagerHandler) Chmod(c *gin.Context) {
	username, ok := h.caller(c)
	if !ok {
		return
	}
	var req struct {
		Path string `json:"path" binding:"required"`
		Mode string `json:"mode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	mode, err := parseMode(req.Mode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err = agent.NewClient(h.agentSock).Call("filemanager.chmod", map[string]interface{}{
		"username": username,
		"path":     req.Path,
		"mode":     mode,
	}, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "permissions updated"})
}

// parseMode accepts "0755" / "755" / "rwxr-xr-x" and returns the
// numeric mode. Octal is the canonical form; the rwx form is convenient
// when round-tripping the value the file list already shows.
func parseMode(s string) (uint32, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty mode")
	}
	// Strip a leading file-type char if the operator pasted "drwxr-xr-x".
	if len(s) == 10 {
		s = s[1:]
	}
	if len(s) == 9 {
		var m uint32
		for i, r := range s {
			bit := uint32(0)
			switch r {
			case 'r':
				bit = 4
			case 'w':
				bit = 2
			case 'x':
				bit = 1
			case '-':
				bit = 0
			default:
				return 0, fmt.Errorf("invalid rwx char %q", r)
			}
			m |= bit << uint((8 - i))
		}
		return m, nil
	}
	// Octal / decimal.
	n, err := strconv.ParseUint(s, 0, 32)
	if err != nil {
		// Try base 8 explicitly if no prefix.
		n, err = strconv.ParseUint(s, 8, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid mode %q", s)
		}
	}
	return uint32(n) & 0777, nil
}

// Copy duplicates a file or directory tree.
func (h *FileManagerHandler) Copy(c *gin.Context) {
	username, ok := h.caller(c)
	if !ok {
		return
	}
	var req struct {
		Src string `json:"src" binding:"required"`
		Dst string `json:"dst" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := agent.NewClient(h.agentSock).Call("filemanager.copy", map[string]interface{}{
		"username": username,
		"src":      req.Src,
		"dst":      req.Dst,
	}, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "copied"})
}

// Compress builds an archive (.zip or .tar.gz) at dst from src.
func (h *FileManagerHandler) Compress(c *gin.Context) {
	username, ok := h.caller(c)
	if !ok {
		return
	}
	var req struct {
		Src string `json:"src" binding:"required"`
		Dst string `json:"dst" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := agent.NewClient(h.agentSock).Call("filemanager.compress", map[string]interface{}{
		"username": username,
		"src":      req.Src,
		"dst":      req.Dst,
	}, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "compressed"})
}

// Extract unpacks a .zip or .tar.gz into a destination directory.
func (h *FileManagerHandler) Extract(c *gin.Context) {
	username, ok := h.caller(c)
	if !ok {
		return
	}
	var req struct {
		Archive string `json:"archive" binding:"required"`
		DstDir  string `json:"dst_dir" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := agent.NewClient(h.agentSock).Call("filemanager.extract", map[string]interface{}{
		"username": username,
		"archive":  req.Archive,
		"dst_dir":  req.DstDir,
	}, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "extracted"})
}
