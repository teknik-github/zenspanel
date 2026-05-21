package handlers

import (
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type FTPHandler struct {
	ftp      *store.FTPAccountStore
	users    *store.UserStore
	packages *store.PackageStore
	homeBase string
	agentSock string
}

func NewFTPHandler(ftp *store.FTPAccountStore, users *store.UserStore, packages *store.PackageStore, homeBase, agentSock string) *FTPHandler {
	return &FTPHandler{ftp: ftp, users: users, packages: packages, homeBase: homeBase, agentSock: agentSock}
}

func (h *FTPHandler) List(c *gin.Context) {
	uid := auth.GetUserID(c)
	accounts, err := h.ftp.ListByUserID(uid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": accounts})
}

func (h *FTPHandler) Create(c *gin.Context) {
	var req struct {
		FTPUsername string `json:"ftp_username" binding:"required"`
		Password    string `json:"password" binding:"required"`
		HomeDir     string `json:"home_dir"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		return
	}

	uid := auth.GetUserID(c)
	user, err := h.users.GetByID(uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Quota check (V61)
	if user.PackageID.Valid {
		pkg, err := h.packages.GetByID(uint64(user.PackageID.Int64))
		if err == nil && pkg.MaxFTPAccounts > 0 {
			count, err := h.ftp.CountByUserID(uid)
			if err == nil && count >= pkg.MaxFTPAccounts {
				c.JSON(http.StatusForbidden, gin.H{"error": "FTP account limit reached for your package"})
				return
			}
		}
	}

	// Default home dir to user's home
	homeDir := req.HomeDir
	if homeDir == "" {
		homeDir = filepath.Join(h.homeBase, user.Username)
	}

	// Hash password for storage
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash password: " + err.Error()})
		return
	}

	// Call agent to provision vsftpd virtual user
	if err := agent.NewClient(h.agentSock).Call("ftp.create", map[string]interface{}{
		"ftp_username":   req.FTPUsername,
		"password":       req.Password,
		"home_dir":       homeDir,
		"panel_username": user.Username,
	}, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "agent: " + err.Error()})
		return
	}

	account := &store.FTPAccount{
		UserID:       uid,
		FTPUsername:  req.FTPUsername,
		PasswordHash: string(hash),
		HomeDir:      homeDir,
		Enabled:      true,
	}
	if err := h.ftp.Create(account); err != nil {
		// Best-effort rollback agent side
		_ = agent.NewClient(h.agentSock).Call("ftp.delete", map[string]interface{}{
			"ftp_username": req.FTPUsername,
		}, nil)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, account)
}

func (h *FTPHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	account, err := h.ftp.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "FTP account not found"})
		return
	}
	if auth.GetRole(c) == "user" && account.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	if err := agent.NewClient(h.agentSock).Call("ftp.delete", map[string]interface{}{
		"ftp_username": account.FTPUsername,
	}, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "agent: " + err.Error()})
		return
	}

	if err := h.ftp.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
