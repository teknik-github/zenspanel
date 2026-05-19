package filebrowser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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
// calls. It exists because `filebrowser config init` creates it
// automatically; with proxy auth the username in the X-Auth-User
// header is the credential, so no password is needed.
const adminUser = "admin"

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

// CreateUser provisions a FileBrowser user record scoped to
// <homeBase>/<username>/. Idempotent: re-creating an existing user
// returns success.
func CreateUser(username, homeBase string) error {
	if err := safe.Username(username); err != nil {
		return err
	}
	scope := homeBase + "/" + username
	body, _ := json.Marshal(modifyUserReq{
		What: "user",
		Data: &fbUser{
			Username: username,
			Password: username + "-no-login", // unused under proxy auth
			Scope:    scope,
			LockPass: false,
			Perm: fbPerm{
				Create: true, Rename: true, Modify: true,
				Delete: true, Share: true, Download: true,
			},
		},
	})

	req, err := http.NewRequest("POST", fileBrowserURL+"/api/users", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	// Proxy auth: the username in this header is the identity
	// FileBrowser will run the request as. Admin scope is required
	// to create users.
	req.Header.Set("X-Auth-User", adminUser)

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
	return fmt.Errorf("filebrowser create user: HTTP %d", resp.StatusCode)
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
	req, err := http.NewRequest("DELETE",
		fileBrowserURL+"/api/users/"+strconv.Itoa(id), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Auth-User", adminUser)
	cli := &http.Client{Timeout: 5 * time.Second}
	resp, err := cli.Do(req)
	if err != nil {
		return fmt.Errorf("filebrowser DELETE /api/users/%d: %w", id, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return fmt.Errorf("filebrowser delete user: HTTP %d", resp.StatusCode)
}

// userID asks FileBrowser for the numeric ID matching a username so we
// can hit the /api/users/<id> DELETE endpoint.
func userID(username string) (int, error) {
	req, _ := http.NewRequest("GET", fileBrowserURL+"/api/users", nil)
	req.Header.Set("X-Auth-User", adminUser)
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
