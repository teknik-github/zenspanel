package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type DatabaseHandler struct {
	databases *store.DatabaseStore
	agentSock string
}

func NewDatabaseHandler(databases *store.DatabaseStore, agentSock string) *DatabaseHandler {
	return &DatabaseHandler{databases: databases, agentSock: agentSock}
}

func (h *DatabaseHandler) List(c *gin.Context) {
	role := auth.GetRole(c)
	userID := auth.GetUserID(c)

	var dbs []store.Database
	var err error
	if role == "admin" {
		if uid := c.Query("user_id"); uid != "" {
			id, _ := strconv.ParseUint(uid, 10, 64)
			dbs, err = h.databases.ListByUserID(id)
		} else {
			dbs, err = h.databases.ListByUserID(0)
		}
	} else {
		dbs, err = h.databases.ListByUserID(userID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": dbs})
}

func (h *DatabaseHandler) Create(c *gin.Context) {
	var req struct {
		DBName     string `json:"db_name" binding:"required"`
		DBUser     string `json:"db_user" binding:"required"`
		DBPassword string `json:"db_password" binding:"required"`
		UserID     uint64 `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := auth.GetUserID(c)
	if auth.GetRole(c) == "admin" && req.UserID > 0 {
		userID = req.UserID
	}

	db := &store.Database{
		UserID: userID,
		DBName: req.DBName,
		DBUser: req.DBUser,
	}
	if err := h.databases.Create(db); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Provision the actual MySQL database + user via the agent. On agent
	// failure we roll back the panel row so the next attempt with the same
	// db_name isn't blocked. The password is never stored in the panel DB —
	// it lives only in MySQL's auth tables and is shown to the caller once.
	agentClient := agent.NewClient(h.agentSock)
	if err := agentClient.Call("mysql.create_database", map[string]interface{}{
		"db_name":     req.DBName,
		"db_user":     req.DBUser,
		"db_password": req.DBPassword,
	}, nil); err != nil {
		_ = h.databases.Delete(db.ID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "provision database: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":          db.ID,
		"user_id":     db.UserID,
		"db_name":     db.DBName,
		"db_user":     db.DBUser,
		"db_password": req.DBPassword,
		"note":        "This password will not be shown again",
	})
}

func (h *DatabaseHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	db, err := h.databases.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "database not found"})
		return
	}
	if auth.GetRole(c) != "admin" && db.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	// Drop the actual MySQL database first. Agent failure is non-fatal — we
	// still delete the panel row, because an orphan row blocks recreate.
	agentClient := agent.NewClient(h.agentSock)
	if err := agentClient.Call("mysql.drop_database", map[string]interface{}{
		"db_name": db.DBName,
		"db_user": db.DBUser,
	}, nil); err != nil {
		log.Printf("mysql.drop_database failed for %s: %v", db.DBName, err)
	}

	if err := h.databases.Delete(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *DatabaseHandler) GetPHPMyAdminToken(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	db, err := h.databases.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "database not found"})
		return
	}
	if auth.GetRole(c) != "admin" && db.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"url":   "/phpmyadmin/",
		"token": db.DBUser,
	})
}
