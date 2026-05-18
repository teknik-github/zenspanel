package handlers

import (
	"net/http"

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
