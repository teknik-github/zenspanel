package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type DatabaseHandler struct {
	databases *store.DatabaseStore
}

func NewDatabaseHandler(databases *store.DatabaseStore) *DatabaseHandler {
	return &DatabaseHandler{databases: databases}
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
	c.JSON(http.StatusCreated, db)
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
