package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zenspanel/zenspanel/internal/store"
)

type AuditLogHandler struct {
	logs *store.AuditLogStore
}

func NewAuditLogHandler(logs *store.AuditLogStore) *AuditLogHandler {
	return &AuditLogHandler{logs: logs}
}

func (h *AuditLogHandler) List(c *gin.Context) {
	filter := store.AuditLogFilter{
		Action:   c.Query("action"),
		DateFrom: c.Query("date_from"),
		DateTo:   c.Query("date_to"),
	}
	if p := c.Query("page"); p != "" {
		filter.Page, _ = strconv.Atoi(p)
	}
	if l := c.Query("limit"); l != "" {
		filter.Limit, _ = strconv.Atoi(l)
	}
	if uid := c.Query("user_id"); uid != "" {
		id, _ := strconv.ParseUint(uid, 10, 64)
		filter.UserID = &id
	}

	logs, total, err := h.logs.List(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": logs, "total": total})
}
