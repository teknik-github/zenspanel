package handlers

import (
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zenspanel/zenspanel/internal/agent"
	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type SSLHandler struct {
	domains   *store.DomainStore
	agentSock string
	leEmail   string
	leStaging bool
}

func NewSSLHandler(domains *store.DomainStore, agentSock, leEmail string, leStaging bool) *SSLHandler {
	return &SSLHandler{
		domains:   domains,
		agentSock: agentSock,
		leEmail:   leEmail,
		leStaging: leStaging,
	}
}

// Issue handles both Let's Encrypt and custom-cert uploads, dispatched on
// the `type` field. The User Panel uses a single endpoint for both flows
// (see frontend/apps/user/src/api/ssl.ts), so we keep that contract.
func (h *SSLHandler) Issue(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	domain, err := h.domains.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	if auth.GetRole(c) != "admin" && domain.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	var req struct {
		Type    string `json:"type" binding:"required"`
		CertPEM string `json:"cert_pem"`
		KeyPEM  string `json:"key_pem"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	agentClient := agent.NewClient(h.agentSock)
	switch req.Type {
	case "letsencrypt":
		// Pre-flight: catch the most common operator misconfigurations
		// before the request even reaches certbot. The ACME server
		// rejects forbidden TLDs with a generic 500 from our agent
		// otherwise — this surfaces a clear 400 instead.
		if msg := validateLEEmail(h.leEmail); msg != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": msg})
			return
		}
		if err := agentClient.Call("ssl.issue_letsencrypt", map[string]interface{}{
			"domain":  domain.Domain,
			"email":   h.leEmail,
			"staging": h.leStaging,
		}, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "issue letsencrypt: " + err.Error()})
			return
		}
		expires := time.Now().AddDate(0, 0, 90).Format("2006-01-02 15:04:05")
		_ = h.domains.Update(domain.ID, map[string]interface{}{
			"ssl_type":       "letsencrypt",
			"ssl_expires_at": expires,
		})
	case "custom":
		if req.CertPEM == "" || req.KeyPEM == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cert_pem and key_pem are required for custom"})
			return
		}
		if err := agentClient.Call("ssl.write_custom_cert", map[string]interface{}{
			"domain":   domain.Domain,
			"cert_pem": req.CertPEM,
			"key_pem":  req.KeyPEM,
		}, nil); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "write custom cert: " + err.Error()})
			return
		}
		fields := map[string]interface{}{"ssl_type": "custom"}
		// Pull NotAfter out of the PEM so the UI can show a real expiry
		// instead of a guess. Failure to parse is non-fatal — we just
		// don't record an expiry, and the UI will show "—".
		if exp, err := certNotAfter(req.CertPEM); err == nil {
			fields["ssl_expires_at"] = exp.Format("2006-01-02 15:04:05")
		}
		_ = h.domains.Update(domain.ID, fields)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be 'letsencrypt' or 'custom'"})
		return
	}

	updated, _ := h.domains.GetByID(domain.ID)
	c.JSON(http.StatusOK, updated)
}

func (h *SSLHandler) Remove(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	domain, err := h.domains.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}
	if auth.GetRole(c) != "admin" && domain.UserID != auth.GetUserID(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	agentClient := agent.NewClient(h.agentSock)
	if err := agentClient.Call("ssl.remove_cert", map[string]interface{}{
		"domain": domain.Domain,
	}, nil); err != nil {
		// Removing the cert files is best-effort — proceed to clear the DB
		// flags so the UI stops reporting an SSL the operator may already
		// have removed by hand.
		c.JSON(http.StatusInternalServerError, gin.H{"error": "remove cert: " + err.Error()})
		return
	}

	_ = h.domains.Update(domain.ID, map[string]interface{}{
		"ssl_type":       "none",
		"ssl_expires_at": nil,
	})
	updated, _ := h.domains.GetByID(domain.ID)
	c.JSON(http.StatusOK, updated)
}

// certNotAfter parses a PEM bundle and returns the NotAfter of the first
// CERTIFICATE block found. The agent already validates the bundle starts
// with BEGIN CERTIFICATE, so anything that lands here should parse — but
// we surface parse errors so the caller can decide whether to record an
// expiry.
func certNotAfter(pemData string) (time.Time, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return time.Time{}, errInvalidPEM
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return time.Time{}, err
	}
	return cert.NotAfter, nil
}

var errInvalidPEM = &pemError{}

type pemError struct{}

func (e *pemError) Error() string { return "invalid PEM" }

// leForbiddenEmailDomains lists email domains the ACME server rejects
// outright when used as an account contact. These are reserved by the
// IANA spec or by Let's Encrypt's own policy. The check is best-effort
// — if a new domain gets added to the deny-list upstream, certbot will
// still surface its own error message; this just catches the obvious
// install-time placeholders before we even round-trip to the agent.
var leForbiddenEmailDomains = map[string]struct{}{
	"example.com":  {},
	"example.net":  {},
	"example.org":  {},
	"test.com":     {},
	"localhost":    {},
	"invalid":      {},
	"localdomain":  {},
}

// validateLEEmail returns a user-facing reason why the configured
// `letsencrypt.email` value won't work, or an empty string if the email
// looks acceptable to ACME. Doesn't validate format strictly — certbot
// will catch that with its own clearer message.
func validateLEEmail(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return "Let's Encrypt email is not configured. Set `letsencrypt.email` in /etc/zenspanel/config.yaml to a real email address you control, then restart the API."
	}
	at := strings.LastIndex(email, "@")
	if at < 1 || at == len(email)-1 {
		return "Let's Encrypt email is malformed. Set `letsencrypt.email` in /etc/zenspanel/config.yaml to a valid address (e.g. admin@yourdomain.com)."
	}
	domain := strings.ToLower(email[at+1:])
	if _, bad := leForbiddenEmailDomains[domain]; bad {
		return "Let's Encrypt rejects '" + domain + "' as a contact email domain. Set `letsencrypt.email` in /etc/zenspanel/config.yaml to a real email address you control (not a reserved/example domain), then restart the API."
	}
	return ""
}
