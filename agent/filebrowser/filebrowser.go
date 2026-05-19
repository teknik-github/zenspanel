package filebrowser

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/zenspanel/zenspanel/agent/safe"
)

// fileBrowserURL is where the locally-running FileBrowser service is
// reachable. We talk to its HTTP API instead of running the CLI because
// the CLI takes an exclusive lock on the SQLite DB that the live service
// already holds — every CLI invocation while the service is running
// times out on the lock.
const fileBrowserURL = "http://127.0.0.1:8081/filebrowser"

// adminUser is the FileBrowser user we authenticate as for management
// calls. Created by install.sh's `users add admin --perm.admin=true`.
const adminUser = "admin"

// FileBrowser proxy auth doesn't make every endpoint trust the
// X-Auth-User header — only /api/login does. To call /api/users we
// must first POST /api/login with X-Auth-User to receive a JWT, then
// pass that JWT in Authorization: Bearer for the actual request.
//
// We cache the JWT for ~1h since FileBrowser's default token TTL is 2h
// and renewing once an hour avoids per-call latency without risking
// expiry races. Cache is process-local — agent restart wipes it, which
// is fine.
type tokenCache struct {
	mu        sync.Mutex
	token     string
	expiresAt time.Time
}

var fbToken tokenCache

func getAdminJWT() (string, error) {
	fbToken.mu.Lock()
	defer fbToken.mu.Unlock()

	if fbToken.token != "" && time.Now().Before(fbToken.expiresAt) {
		return fbToken.token, nil
	}

	// /api/login under proxy auth: the body can be empty; the
	// X-Auth-User header IS the credential. Response body is the raw
	// JWT string (not JSON-wrapped).
	req, err := http.NewRequest("POST", fileBrowserURL+"/api/login", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Auth-User", adminUser)

	cli := &http.Client{Timeout: 5 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return "", fmt.Errorf("filebrowser login: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("filebrowser login HTTP %d: %s", resp.StatusCode, string(body))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	tok := string(bytes.TrimSpace(body))
	if tok == "" {
		return "", fmt.Errorf("filebrowser login returned empty token")
	}
	fbToken.token = tok
	fbToken.expiresAt = time.Now().Add(1 * time.Hour)
	return tok, nil
}

// authReq builds an *http.Request with the cached admin JWT attached.
// Used by all management calls below.
func authReq(method, url string, body io.Reader) (*http.Request, error) {
	tok, err := getAdminJWT()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Auth", tok)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

type fbUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Scope    string `json:"scope"`
	LockPass bool   `json:"lockPassword"`
	Perm     fbPerm `json:"perm"`
}

type fbPerm struct {
	Admin    bool `json:"admin"`
	Execute  bool `json:"execute"`
	Create   bool `json:"create"`
	Rename   bool `json:"rename"`
	Modify   bool `json:"modify"`
	Delete   bool `json:"delete"`
	Share    bool `json:"share"`
	Download bool `json:"download"`
}

type modifyUserReq struct {
	What string  `json:"what"`
	Data *fbUser `json:"data"`
}

// CreateUser provisions a FileBrowser user record scoped to the
// username's directory under FileBrowser's configured Root. The scope
// is the username itself (relative), not the absolute path: FileBrowser
// joins Root + scope internally, and an absolute scope would
// double-prefix into a path that doesn't exist on disk.
func CreateUser(username, homeBase string) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	_ = homeBase // kept in signature for symmetry with the other agent.* calls

	// FileBrowser enforces a 12-char minimum on the password column
	// even under proxy auth where the password is never used. We
	// generate a 32-char random hex string so any username length is
	// safe without leaking a guessable pattern.
	pw := make([]byte, 16)
	if _, err := rand.Read(pw); err != nil {
		return fmt.Errorf("rand: %w", err)
	}
	password := hex.EncodeToString(pw)

	body, _ := json.Marshal(modifyUserReq{
		What: "user",
		Data: &fbUser{
			Username: username,
			Password: password, // unused under proxy auth, just satisfies the column
			Scope:    username, // relative to Root
			LockPass: false,
			Perm: fbPerm{
				Create: true, Rename: true, Modify: true,
				Delete: true, Share: true, Download: true,
			},
		},
	})

	req, err := authReq("POST", fileBrowserURL+"/api/users", bytes.NewReader(body))
	if err != nil {
		return err
	}

	cli := &http.Client{Timeout: 5 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("filebrowser POST /api/users: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode == http.StatusConflict {
		return nil // user already exists — exactly what we want
	}
	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("filebrowser create user: HTTP %d: %s", resp.StatusCode, string(respBody))
}

// DeleteUser removes the FileBrowser user record. Looks the user up
// first because the API expects an integer ID, not a username.
func DeleteUser(username string) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	id, err := userID(username)
	if err != nil {
		return err
	}
	if id == 0 {
		return nil // nothing to delete
	}
	req, err := authReq("DELETE", fileBrowserURL+"/api/users/"+strconv.Itoa(id), nil)
	if err != nil {
		return err
	}
	cli := &http.Client{Timeout: 5 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("filebrowser DELETE /api/users/%d: %w", id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("filebrowser delete user: HTTP %d: %s", resp.StatusCode, string(respBody))
}

// userID asks FileBrowser for the numeric ID matching a username so we
// can hit the /api/users/<id> DELETE endpoint.
func userID(username string) (int, error) {
	req, err := authReq("GET", fileBrowserURL+"/api/users", nil)
	if err != nil {
		return 0, err
	}
	cli := &http.Client{Timeout: 5 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return 0, fmt.Errorf("list users: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("list users: HTTP %d", resp.StatusCode)
	}
	var users []struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return 0, err
	}
	for _, u := range users {
		if u.Username == username {
			return u.ID, nil
		}
	}
	return 0, nil
}
