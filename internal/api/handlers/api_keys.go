package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type APIKeyHandler struct {
	apiKeys *store.APIKeyStore
}

func NewAPIKeyHandler(apiKeys *store.APIKeyStore) *APIKeyHandler {
	return &APIKeyHandler{apiKeys: apiKeys}
}

func (h *APIKeyHandler) List(c *gin.Context) {
	keys, err := h.apiKeys.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// never return key_hash
	type safeKey struct {
		ID          uint64 `json:"id"`
		Name        string `json:"name"`
		KeyPrefix   string `json:"key_prefix"`
		Permissions string `json:"permissions"`
		LastUsedAt  interface{} `json:"last_used_at"`
		ExpiresAt   interface{} `json:"expires_at"`
		CreatedAt   interface{} `json:"created_at"`
	}
	safe := make([]safeKey, len(keys))
	for i, k := range keys {
		safe[i] = safeKey{
			ID:          k.ID,
			Name:        k.Name,
			KeyPrefix:   k.KeyPrefix,
			Permissions: k.Permissions,
			LastUsedAt:  k.LastUsedAt,
			ExpiresAt:   k.ExpiresAt,
			CreatedAt:   k.CreatedAt,
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": safe})
}

func (h *APIKeyHandler) Create(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Permissions string `json:"permissions" binding:"required"`
		ExpiresAt   string `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// generate random key: zp_live_ + 32 hex chars
	raw := make([]byte, 16)
	rand.Read(raw)
	fullKey := "zp_live_" + hex.EncodeToString(raw)

	hash, err := store.HashAPIKey(fullKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash failed"})
		return
	}

	key := &store.APIKey{
		Name:        req.Name,
		KeyHash:     hash,
		KeyPrefix:   fullKey[:8],
		Permissions: req.Permissions,
		CreatedBy:   auth.GetUserID(c),
	}

	if err := h.apiKeys.Create(key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":     key.ID,
		"key":    fullKey,
		"prefix": key.KeyPrefix,
		"note":   "This key will not be shown again",
	})
}

func (h *APIKeyHandler) Revoke(c *gin.Context) {
	var req struct {
		ID uint64 `uri:"id" binding:"required"`
	}
	if err := c.ShouldBindUri(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.apiKeys.Delete(req.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "revoked"})
}
