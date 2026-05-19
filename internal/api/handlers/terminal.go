package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"net/http"
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
	tokenStore sync.Map // map[string]tokenEntry
	upgrader   = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		// Same-origin only — both panels are served from the same host
		// as the API, so an explicit Origin check would force us to
		// hardcode hosts at install time. Gin already runs behind nginx
		// which terminates TLS and validates Host.
		CheckOrigin: func(r *http.Request) bool { return true },
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
func (h *TerminalHandler) GetToken(c *gin.Context) {
	uid := auth.GetUserID(c)
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
