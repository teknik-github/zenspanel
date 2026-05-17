package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zenspanel/zenspanel/internal/store"
)

type PHPVersionHandler struct {
	phpVersions *store.PHPVersionStore
}

func NewPHPVersionHandler(phpVersions *store.PHPVersionStore) *PHPVersionHandler {
	return &PHPVersionHandler{phpVersions: phpVersions}
}

func (h *PHPVersionHandler) List(c *gin.Context) {
	versions, err := h.phpVersions.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": versions})
}

func (h *PHPVersionHandler) ListEnabled(c *gin.Context) {
	versions, err := h.phpVersions.ListEnabled()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": versions})
}

func (h *PHPVersionHandler) Enable(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.phpVersions.SetEnabled(id, true); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "enabled"})
}

func (h *PHPVersionHandler) Disable(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.phpVersions.SetEnabled(id, false); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "disabled"})
}
