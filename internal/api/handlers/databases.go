package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type DatabaseHandler struct {
	databases *store.DatabaseStore
	agentSock string
	redis     *redis.Client // nil = phpMyAdmin SSO disabled
}

func NewDatabaseHandler(databases *store.DatabaseStore, agentSock string, rdb *redis.Client) *DatabaseHandler {
	return &DatabaseHandler{databases: databases, agentSock: agentSock, redis: rdb}
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
	if auth.GetRole(c) == "user" && db.UserID != auth.GetUserID(c) {
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

// ResetPassword generates a new random password for the database user,
// updates MySQL via the agent, and returns the new password once (V56).
// The password is never stored — the user must copy it immediately.
func (h *DatabaseHandler) ResetPassword(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	db, err := h.databases.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "database not found"})
		return
	}
	if auth.GetRole(c) == "user" && db.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	// Generate a strong random password (16 bytes = 32 hex chars).
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rand: " + err.Error()})
		return
	}
	// Use alphanumeric chars from the safe charset so the password is
	// valid for MySQL's IDENTIFIED BY clause without escaping.
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	newPassword := make([]byte, 16)
	for i, v := range b {
		newPassword[i] = chars[int(v)%len(chars)]
	}
	pwd := string(newPassword)

	if err := agent.NewClient(h.agentSock).Call("mysql.reset_password", map[string]interface{}{
		"db_user":      db.DBUser,
		"new_password": pwd,
	}, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "reset password: " + err.Error()})
		return
	}

	// Return the new password once — it is never stored (V56).
	c.JSON(http.StatusOK, gin.H{
		"db_user":      db.DBUser,
		"new_password": pwd,
	})
}
// frontend that still calls /databases/:id/phpmyadmin. It now hands back
// a launch URL that the user can open to bounce through the SSO flow.
func (h *DatabaseHandler) GetPHPMyAdminToken(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	db, err := h.databases.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "database not found"})
		return
	}
	if auth.GetRole(c) == "user" && db.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"url": fmt.Sprintf("/api/v1/databases/%d/phpmyadmin/launch", db.ID),
	})
}

// pmaSSOPayload is what we stash in Redis between Launch and Redeem. The
// password is whatever ResetUserPassword just generated; it lives only in
// Redis and is wiped after the first redeem.
type pmaSSOPayload struct {
	DBUser   string `json:"db_user"`
	Password string `json:"password"`
}

// pmaSSOPasswordChars matches agent/safe.dbPasswordRe so the freshly-
// generated password passes the agent's validator. We avoid characters
// that would need HTML-escaping in the auto-submit form's value attribute
// (the html/template package handles escaping anyway, but keeping the
// charset boring keeps the SQL identifier path safe too).
const pmaSSOPasswordChars = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"

func generatePMAPassword() (string, error) {
	const n = 24
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = pmaSSOPasswordChars[int(b[i])%len(pmaSSOPasswordChars)]
	}
	return string(b), nil
}

func generatePMAToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// LaunchPHPMyAdmin starts the SSO flow. It resets the database user's
// MySQL password to a freshly-generated value, stashes that value plus
// the username in Redis under a 60-second one-time token, then redirects
// the browser to the redeem endpoint. The redeem endpoint serves an
// auto-submitting form that POSTs into phpMyAdmin's cookie-auth login.
//
// Resetting the password every launch is the cost of not storing
// passwords in the panel DB. The user gets a fresh logged-in session;
// the old password (if anyone remembers it) stops working.
func (h *DatabaseHandler) LaunchPHPMyAdmin(c *gin.Context) {
	if h.redis == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "phpMyAdmin SSO requires Redis"})
		return
	}
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	db, err := h.databases.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "database not found"})
		return
	}
	if auth.GetRole(c) == "user" && db.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	password, err := generatePMAPassword()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate password"})
		return
	}
	token, err := generatePMAToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate token"})
		return
	}

	// Reset the MySQL password via the agent. The agent validates both
	// db_user and the new password against agent/safe before issuing
	// ALTER USER, so we don't need to re-validate here.
	agentClient := agent.NewClient(h.agentSock)
	if err := agentClient.Call("mysql.reset_password", map[string]interface{}{
		"db_user":      db.DBUser,
		"new_password": password,
	}, nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "reset password: " + err.Error()})
		return
	}

	payload, _ := json.Marshal(pmaSSOPayload{DBUser: db.DBUser, Password: password})
	if err := h.redis.Set(c.Request.Context(), "pma_sso:"+token, payload, 60*time.Second).Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "store token: " + err.Error()})
		return
	}

	// Return the redeem URL as JSON. The frontend window.open()s it in a
	// new tab; the redeem page (no JWT required) auto-submits a form into
	// phpMyAdmin's cookie auth.
	c.JSON(http.StatusOK, gin.H{
		"url": "/api/v1/phpmyadmin/sso/" + token,
	})
}

// pmaRedeemTmpl is the auto-submitting form served at the redeem URL.
// Two important properties:
//   - The form posts to /phpmyadmin/index.php with phpMyAdmin's cookie-
//     auth field names (pma_username, pma_password, server). phpMyAdmin
//     accepts that POST as a successful login and sets its own auth
//     cookie before serving the dashboard.
//   - html/template auto-escapes the credential values so a username or
//     password containing quote characters can't break out of the value
//     attribute. The agent's safe.DBPassword regex already excludes the
//     dangerous characters, but defence in depth here is free.
var pmaRedeemTmpl = template.Must(template.New("pma_sso").Parse(`<!doctype html>
<html><head><meta charset="utf-8"><title>Opening phpMyAdmin...</title>
<style>body{font-family:system-ui,sans-serif;background:#f9fafb;color:#374151;display:flex;align-items:center;justify-content:center;height:100vh;margin:0;font-size:14px}</style>
</head><body>
<form id="f" method="POST" action="/phpmyadmin/" autocomplete="off">
  <input type="hidden" name="pma_username" value="{{.User}}">
  <input type="hidden" name="pma_password" value="{{.Password}}">
  <input type="hidden" name="server" value="1">
  <input type="hidden" name="target" value="index.php">
  <input type="hidden" name="set_session" value="">
  <noscript>JavaScript is required for phpMyAdmin auto-login. <button type="submit">Continue</button></noscript>
</form>
<p>Opening phpMyAdmin...</p>
<script>
// Try the standard login endpoint first; fall back to index.php if needed.
var f = document.getElementById('f');
f.submit();
</script>
</body></html>`))

// RedeemPHPMyAdmin consumes a one-time SSO token, stores credentials in
// a short-lived Redis key readable by the PHP bridge, then redirects the
// browser to the PHP bridge which sets the PHP session and redirects to
// phpMyAdmin. The token is deleted from Redis on first read so a reused
// or leaked token returns 410 Gone. No JWT required — the token itself
// is the credential, scoped to ~60 seconds and one redemption.
func (h *DatabaseHandler) RedeemPHPMyAdmin(c *gin.Context) {
	if h.redis == nil {
		c.String(http.StatusServiceUnavailable, "phpMyAdmin SSO requires Redis")
		return
	}
	token := c.Param("token")
	if len(token) != 32 || !isHex(token) {
		c.String(http.StatusBadRequest, "invalid token")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	key := "pma_sso:" + token
	// Use GET + DEL instead of GETDEL for Redis < 6.2 compatibility.
	val, err := h.redis.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		c.String(http.StatusGone, "token expired or already used")
		return
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "redis: "+err.Error())
		return
	}
	// Delete immediately — one-time use.
	_ = h.redis.Del(ctx, key).Err()

	var p pmaSSOPayload
	if err := json.Unmarshal([]byte(val), &p); err != nil {
		c.String(http.StatusInternalServerError, "decode payload")
		return
	}

	// Store credentials in a bridge key that the PHP signon.php script
	// reads to set the PHP session. Use a new short-lived token so the
	// credentials are only exposed for the duration of the redirect.
	bridgeToken := make([]byte, 16)
	if _, err := rand.Read(bridgeToken); err != nil {
		c.String(http.StatusInternalServerError, "rand")
		return
	}
	bridgeKey := "pma_bridge:" + hex.EncodeToString(bridgeToken)
	bridgePayload, _ := json.Marshal(p)
	_ = h.redis.Set(ctx, bridgeKey, bridgePayload, 30*time.Second).Err()

	// Redirect to the PHP bridge which sets the PHP session and then
	// redirects to phpMyAdmin. The bridge token is the only credential
	// in the URL — it's one-time-use and expires in 30 seconds.
	c.Redirect(http.StatusFound, "/phpmyadmin/signon.php?bridge="+hex.EncodeToString(bridgeToken))
}

// isHex checks that every byte is 0-9 or a-f. The token is generated with
// hex.EncodeToString, so anything else is automatically suspect.
func isHex(s string) bool {
	for _, r := range s {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return strings.TrimSpace(s) != ""
}
