package handlers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	agentclient "github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/store"
)

type BackupTargetHandler struct {
	targets   *store.BackupTargetStore
	agentSock string
	encKey    []byte // AES-256-GCM key for secret encryption (V47)
}

func NewBackupTargetHandler(targets *store.BackupTargetStore, agentSock, jwtSecret, encKeyHex string) *BackupTargetHandler {
	var key []byte
	if encKeyHex != "" {
		key, _ = hex.DecodeString(encKeyHex)
	}
	if len(key) != 32 {
		h := []byte(jwtSecret)
		for len(h) < 32 {
			h = append(h, h...)
		}
		key = h[:32]
	}
	return &BackupTargetHandler{targets: targets, agentSock: agentSock, encKey: key}
}

func (h *BackupTargetHandler) encryptSecret(secret string) (string, error) {
	block, err := aes.NewCipher(h.encKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nonce, nonce, []byte(secret), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

func (h *BackupTargetHandler) List(c *gin.Context) {
	targets, err := h.targets.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": targets})
}

func (h *BackupTargetHandler) Create(c *gin.Context) {
	var req struct {
		Name      string `json:"name"       binding:"required"`
		Type      string `json:"type"`
		Bucket    string `json:"bucket"     binding:"required"`
		Prefix    string `json:"prefix"`
		AccessKey string `json:"access_key" binding:"required"`
		SecretKey string `json:"secret_key" binding:"required"`
		Region    string `json:"region"`
		Endpoint  string `json:"endpoint"`
		Enabled   *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	enc, err := h.encryptSecret(req.SecretKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encrypt secret: " + err.Error()})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	if req.Type == "" {
		req.Type = "s3"
	}
	if req.Region == "" {
		req.Region = "us-east-1"
	}

	t := &store.BackupTarget{
		Name:         req.Name,
		Type:         req.Type,
		Bucket:       req.Bucket,
		Prefix:       req.Prefix,
		AccessKey:    req.AccessKey,
		SecretKeyEnc: enc,
		Region:       req.Region,
		Endpoint:     req.Endpoint,
		Enabled:      enabled,
	}
	if err := h.targets.Create(t); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, t)
}

func (h *BackupTargetHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	existing, err := h.targets.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "target not found"})
		return
	}

	var req struct {
		Name      string `json:"name"`
		Type      string `json:"type"`
		Bucket    string `json:"bucket"`
		Prefix    string `json:"prefix"`
		AccessKey string `json:"access_key"`
		SecretKey string `json:"secret_key"` // empty = keep existing
		Region    string `json:"region"`
		Endpoint  string `json:"endpoint"`
		Enabled   *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Type != "" {
		existing.Type = req.Type
	}
	if req.Bucket != "" {
		existing.Bucket = req.Bucket
	}
	existing.Prefix = req.Prefix
	if req.AccessKey != "" {
		existing.AccessKey = req.AccessKey
	}
	if req.SecretKey != "" {
		enc, err := h.encryptSecret(req.SecretKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "encrypt secret: " + err.Error()})
			return
		}
		existing.SecretKeyEnc = enc
	}
	if req.Region != "" {
		existing.Region = req.Region
	}
	existing.Endpoint = req.Endpoint
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}

	if err := h.targets.Update(id, existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *BackupTargetHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.targets.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// Test verifies connectivity to the backup target by asking the agent
// to list the bucket root via rclone.
func (h *BackupTargetHandler) Test(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	t, err := h.targets.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "target not found"})
		return
	}

	if err := agentclient.NewClient(h.agentSock).Call("backup.test_s3", map[string]interface{}{
		"target_id":      t.ID,
		"name":           t.Name,
		"type":           t.Type,
		"bucket":         t.Bucket,
		"prefix":         t.Prefix,
		"access_key":     t.AccessKey,
		"secret_key_enc": t.SecretKeyEnc,
		"region":         t.Region,
		"endpoint":       t.Endpoint,
	}, nil); err != nil {
		c.JSON(http.StatusOK, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
