package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

// Token store. Browsers can't attach Authorization headers when opening
// a WebSocket from page JS, so the standard pattern is: mint a one-time,
// short-lived token over the JWT-protected REST endpoint, then redeem it
// in the WebSocket query string. The token IS the credential.
type tokenEntry struct {
	username  string
	expiresAt time.Time
}

var (
	tokenStore     sync.Map // map[string]tokenEntry
	tokenRateStore sync.Map // map[uint64]time.Time — last mint time per userID
	upgrader   = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// Same-origin check (V14). Compare Origin to the canonical host.
		// When behind nginx, r.Host is the upstream address (127.0.0.1:8080)
		// but the browser sends Origin with the public host. We check both
		// r.Host and X-Forwarded-Host so the check works whether the API is
		// accessed directly or through a reverse proxy.
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // non-browser client (curl, native WS)
			}
			// Collect candidate hosts: direct + forwarded.
			hosts := []string{r.Host}
			if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
				hosts = append(hosts, fwd)
			}
			for _, h := range hosts {
				if origin == "http://"+h || origin == "https://"+h ||
					strings.HasSuffix(origin, "://"+h) {
					return true
				}
			}
			return false
		},
	}
)

type TerminalHandler struct {
	users     *store.UserStore
	agentSock string
}

func NewTerminalHandler(users *store.UserStore, agentSock string) *TerminalHandler {
	return &TerminalHandler{users: users, agentSock: agentSock}
}

// GetToken mints a one-time terminal token for the authenticated user.
// Caller must have terminal_enabled — package is a soft feature flag and
// admins toggle it per-user. Token TTL is 60s; the WS handshake should
// happen immediately after this returns.
//
// Rate-limited per-user (V17) to 1 mint per 5 seconds. Cheap defence
// against an attacker who got a stolen JWT and is trying to spawn a
// shell mid-session — without the limiter they could keep retrying
// against any future race in the redeem path. 5 s is well above the
// time a legitimate user needs (token → ws upgrade is sub-second).
func (h *TerminalHandler) GetToken(c *gin.Context) {
	uid := auth.GetUserID(c)
	if !checkTokenRate(uid) {
		c.Header("Retry-After", "5")
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many token requests, wait 5 seconds"})
		return
	}
	user, err := h.users.GetByID(uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if !user.TerminalEnabled {
		c.JSON(http.StatusForbidden, gin.H{"error": "terminal not enabled for this account"})
		return
	}

	tb := make([]byte, 16)
	if _, err := rand.Read(tb); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rand: " + err.Error()})
		return
	}
	token := hex.EncodeToString(tb)
	tokenStore.Store(token, tokenEntry{
		username:  user.Username,
		expiresAt: time.Now().Add(60 * time.Second),
	})
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// AdminGetToken mints a terminal token for an admin session. The admin
// can specify a target username to shell into that user's account, or
// leave it empty to get a shell as the zenspanel system user (V48 —
// never spawns a root shell; zenspanel user has no sudo).
func (h *TerminalHandler) AdminGetToken(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
	}
	_ = c.ShouldBindJSON(&req)

	// Default to the zenspanel system user if no target specified.
	targetUsername := req.Username
	if targetUsername == "" {
		targetUsername = "zenspanel"
	}

	// If a specific user is requested, verify they exist.
	if req.Username != "" {
		if _, err := h.users.GetByUsername(req.Username); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}
	}

	tb := make([]byte, 16)
	if _, err := rand.Read(tb); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rand: " + err.Error()})
		return
	}
	token := hex.EncodeToString(tb)
	tokenStore.Store(token, tokenEntry{
		username:  targetUsername,
		expiresAt: time.Now().Add(60 * time.Second),
	})
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// checkTokenRate is a per-user 5-second sliding-window check. Returns
// true if the caller may mint a token now and updates the timestamp;
// false if a previous mint is still inside the cooldown.
//
// In-process map — fine for single-API-instance deploys. Multi-instance
// deploys should swap this for Redis (same Lua trick the login limiter
// uses), but ZensPanel's terminal is per-server anyway.
func checkTokenRate(userID uint64) bool {
	now := time.Now()
	if last, ok := tokenRateStore.Load(userID); ok {
		if now.Sub(last.(time.Time)) < 5*time.Second {
			return false
		}
	}
	tokenRateStore.Store(userID, now)
	return true
}

// redeemToken does a one-time, atomic load+delete plus an expiry check.
// LoadAndDelete is the right primitive: race-free, can't be replayed.
func redeemToken(token string) (tokenEntry, bool) {
	v, ok := tokenStore.LoadAndDelete(token)
	if !ok {
		return tokenEntry{}, false
	}
	entry := v.(tokenEntry)
	if time.Now().After(entry.expiresAt) {
		return tokenEntry{}, false
	}
	return entry, true
}

// Connect upgrades the request to a WebSocket and bridges the browser to
// a fresh PTY hosted by the agent. Wire format toward the browser:
//
//	{"type":"output","data":"<base64 of raw bytes>"}
//	{"type":"input","data":"<utf-8 string>"}
//
// Toward the PTY we just write the raw input bytes. Output bytes get
// base64-encoded because xterm.js + JSON is happier with text.
func (h *TerminalHandler) Connect(c *gin.Context) {
	token := c.Query("token")
	entry, ok := redeemToken(token)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	// Ask the agent to spawn a PTY and expose it on a one-shot Unix
	// socket. We get back the socket path; we dial it ourselves.
	var spawnRes struct {
		SockPath string `json:"sock_path"`
	}
	if err := agent.NewClient(h.agentSock).Call("terminal.stream", map[string]interface{}{
		"username": entry.username,
	}, &spawnRes); err != nil {
		log.Printf("terminal.stream failed for user %s: %v", entry.username, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "spawn pty: " + err.Error()})
		return
	}

	ptyConn, err := net.Dial("unix", spawnRes.SockPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "dial pty: " + err.Error()})
		return
	}

	// Upgrade comes AFTER the spawn so a spawn failure can return JSON
	// instead of being lost inside an already-upgraded connection.
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		ptyConn.Close()
		return
	}

	// Bridge. Two goroutines so either side closing tears the other one
	// down. We close ws + ptyConn at the end of the handler so deferred
	// io.Copy(s) unblock.
	defer ws.Close()
	defer ptyConn.Close()

	// PTY → WS
	done := make(chan struct{}, 2)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptyConn.Read(buf)
			if n > 0 {
				msg, _ := json.Marshal(map[string]string{
					"type": "output",
					"data": base64.StdEncoding.EncodeToString(buf[:n]),
				})
				if werr := ws.WriteMessage(websocket.TextMessage, msg); werr != nil {
					break
				}
			}
			if err != nil {
				if err != io.EOF {
					_ = ws.WriteMessage(websocket.TextMessage, []byte(`{"type":"output","data":""}`))
				}
				break
			}
		}
		done <- struct{}{}
	}()

	// WS → PTY
	go func() {
		for {
			_, raw, err := ws.ReadMessage()
			if err != nil {
				break
			}
			var msg struct {
				Type string `json:"type"`
				Data string `json:"data"`
			}
			if err := json.Unmarshal(raw, &msg); err != nil {
				continue
			}
			if msg.Type != "input" {
				continue
			}
			if _, err := ptyConn.Write([]byte(msg.Data)); err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	<-done
}

// init starts a background pruner so the token map doesn't grow
// unboundedly when tokens are minted but never redeemed (eg. user
// closes the tab right after clicking Terminal).
func init() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			now := time.Now()
			tokenStore.Range(func(k, v interface{}) bool {
				if now.After(v.(tokenEntry).expiresAt) {
					tokenStore.Delete(k)
				}
				return true
			})
		}
	}()
}
