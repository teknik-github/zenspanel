package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	agentclient "github.com/zenspanel/zenspanel/internal/agent"
	agentcron "github.com/zenspanel/zenspanel/agent/cron"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type CronJobHandler struct {
	crons    *store.CronJobStore
	users    *store.UserStore
	packages *store.PackageStore
	agentSock string
}

func NewCronJobHandler(crons *store.CronJobStore, users *store.UserStore, packages *store.PackageStore, agentSock string) *CronJobHandler {
	return &CronJobHandler{crons: crons, users: users, packages: packages, agentSock: agentSock}
}

// syncCrontab rebuilds the user's full crontab from DB and pushes it to
// the agent. Called after every mutation so the crontab stays in sync.
func (h *CronJobHandler) syncCrontab(userID uint64, username string) error {
	jobs, err := h.crons.ListByUserID(userID)
	if err != nil {
		return err
	}
	agentJobs := make([]agentcron.Job, len(jobs))
	for i, j := range jobs {
		agentJobs[i] = agentcron.Job{
			Expression: j.Expression,
			Command:    j.Command,
			Enabled:    j.Enabled,
		}
	}
	return agentclient.NewClient(h.agentSock).Call("cron.sync", map[string]interface{}{
		"username": username,
		"jobs":     agentJobs,
	}, nil)
}

func (h *CronJobHandler) List(c *gin.Context) {
	userID := auth.GetUserID(c)
	jobs, err := h.crons.ListByUserID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": jobs})
}

func (h *CronJobHandler) Create(c *gin.Context) {
	userID := auth.GetUserID(c)
	var req struct {
		Expression string `json:"expression" binding:"required"`
		Command    string `json:"command"    binding:"required"`
		Enabled    *bool  `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate expression + command via agent package (V23, V24).
	if err := agentcron.ValidateExpression(req.Expression); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := agentcron.ValidateCommand(req.Command); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Quota check (V25): count existing jobs vs package limit.
	user, err := h.users.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup user"})
		return
	}
	if user.PackageID.Valid {
		pkg, err := h.packages.GetByID(uint64(user.PackageID.Int64))
		if err == nil && pkg.MaxCronJobs > 0 {
			count, _ := h.crons.CountByUserID(userID)
			if count >= pkg.MaxCronJobs {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": "cron job quota reached for your package"})
				return
			}
		}
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	job := &store.CronJob{
		UserID:     userID,
		Expression: req.Expression,
		Command:    req.Command,
		Enabled:    enabled,
	}
	if err := h.crons.Create(job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := h.syncCrontab(userID, user.Username); err != nil {
		c.JSON(http.StatusCreated, gin.H{"data": job, "warning": "crontab sync: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": job})
}

func (h *CronJobHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := auth.GetUserID(c)

	job, err := h.crons.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cron job not found"})
		return
	}
	if auth.GetRole(c) == "user" && job.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var fields map[string]interface{}
	if err := c.ShouldBindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate updated fields before writing (V23, V24).
	if expr, ok := fields["expression"].(string); ok {
		if err := agentcron.ValidateExpression(expr); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}
	if cmd, ok := fields["command"].(string); ok {
		if err := agentcron.ValidateCommand(cmd); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	if err := h.crons.Update(id, fields); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user, _ := h.users.GetByID(job.UserID)
	if user != nil {
		_ = h.syncCrontab(job.UserID, user.Username)
	}
	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *CronJobHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	userID := auth.GetUserID(c)

	job, err := h.crons.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "cron job not found"})
		return
	}
	if auth.GetRole(c) == "user" && job.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	if err := h.crons.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user, _ := h.users.GetByID(job.UserID)
	if user != nil {
		_ = h.syncCrontab(job.UserID, user.Username)
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
