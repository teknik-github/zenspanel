package handlers

import (
	"net/http"
	"net/netip"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/store"
)

type IPAllowlistHandler struct {
	store     *store.AdminAllowedIPStore
	agentSock string
}

func NewIPAllowlistHandler(store *store.AdminAllowedIPStore, agentSock string) *IPAllowlistHandler {
	return &IPAllowlistHandler{store: store, agentSock: agentSock}
}

func (h *IPAllowlistHandler) List(c *gin.Context) {
	rows, err := h.store.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows})
}

func (h *IPAllowlistHandler) Create(c *gin.Context) {
	var req struct {
		IPCIDR string `json:"ip_cidr" binding:"required"`
		Note   string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate IP or CIDR
	if _, err := netip.ParseAddr(req.IPCIDR); err != nil {
		if _, err2 := netip.ParsePrefix(req.IPCIDR); err2 != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid IP address or CIDR: " + req.IPCIDR})
			return
		}
	}

	row, err := h.store.Create(req.IPCIDR, req.Note)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.syncNginx(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "nginx sync: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, row)
}

func (h *IPAllowlistHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.store.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := h.syncNginx(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "nginx sync: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// syncNginx reads current allowlist from DB and pushes to nginx agent.
func (h *IPAllowlistHandler) syncNginx() error {
	rows, err := h.store.List()
	if err != nil {
		return err
	}
	ips := make([]string, len(rows))
	for i, r := range rows {
		ips[i] = r.IPCIDR
	}
	return agent.NewClient(h.agentSock).Call("nginx.set_admin_allowlist", map[string]interface{}{
		"ips": ips,
	}, nil)
}
