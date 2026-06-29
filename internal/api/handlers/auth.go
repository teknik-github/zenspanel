package handlers

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/zenspanel/zenspanel/internal/auth"
	"github.com/zenspanel/zenspanel/internal/store"
)

type AuthHandler struct {
	users   *store.UserStore
	secret  string
	expiry  string
	totpKey []byte // 32-byte AES-256-GCM key for TOTP secret encryption (V27)
}

func NewAuthHandler(users *store.UserStore, secret, expiry, totpKeyHex string) *AuthHandler {
	var key []byte
	if totpKeyHex != "" {
		k, err := hex.DecodeString(totpKeyHex)
		if err == nil && len(k) == 32 {
			key = k
		}
	}
	// If no key configured, derive a deterministic one from the JWT secret
	// so existing installs work without config changes. Not ideal but safe
	// enough — the secret is already protecting the JWT.
	if key == nil {
		h := []byte(secret)
		for len(h) < 32 {
			h = append(h, h...)
		}
		key = h[:32]
	}
	return &AuthHandler{users: users, secret: secret, expiry: expiry, totpKey: key}
}

func qrImageDataURL(img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}

	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func isSecureRequest(c *gin.Context) bool {
	return c.Request.TLS != nil ||
		strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}

func setAuthCookie(c *gin.Context, token string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("zenspanel_token", token, 24*60*60, "/", "", isSecureRequest(c), true)
}

func clearAuthCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("zenspanel_token", "", -1, "/", "", isSecureRequest(c), true)
}

// tempTokenStore holds short-lived tokens issued after password auth when
// 2FA is required. The token is redeemed by /auth/2fa/verify (V28).
var tempTokenStore sync.Map // map[string]tempTokenEntry

type tempTokenEntry struct {
	userID    uint64
	expiresAt time.Time
}

func mintTempToken(userID uint64) string {
	b := make([]byte, 16)
	rand.Read(b)
	tok := hex.EncodeToString(b)
	tempTokenStore.Store(tok, tempTokenEntry{userID: userID, expiresAt: time.Now().Add(5 * time.Minute)})
	return tok
}

func redeemTempToken(tok string) (uint64, bool) {
	v, ok := tempTokenStore.LoadAndDelete(tok)
	if !ok {
		return 0, false
	}
	e := v.(tempTokenEntry)
	if time.Now().After(e.expiresAt) {
		return 0, false
	}
	return e.userID, true
}

// encryptTOTP encrypts a TOTP secret with AES-256-GCM (V27).
func (h *AuthHandler) encryptTOTP(secret string) (string, error) {
	block, err := aes.NewCipher(h.totpKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := gcm.Seal(nonce, nonce, []byte(secret), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

// decryptTOTP decrypts a TOTP secret encrypted by encryptTOTP.
func (h *AuthHandler) decryptTOTP(enc string) (string, error) {
	ct, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(h.totpKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(ct) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}
	plain, err := gcm.Open(nil, ct[:gcm.NonceSize()], ct[gcm.NonceSize():], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.users.GetByUsername(req.Username)
	if err != nil || !h.users.CheckPassword(user.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if user.Status == "suspended" {
		c.JSON(http.StatusForbidden, gin.H{"error": "account suspended"})
		return
	}

	// If 2FA is enabled, return a short-lived temp token instead of the
	// full JWT. The client must POST to /auth/2fa/verify with the TOTP
	// code to get the real token (V28).
	if user.TOTPEnabled {
		tempTok := mintTempToken(user.ID)
		c.JSON(http.StatusOK, gin.H{
			"requires_2fa": true,
			"temp_token":   tempTok,
		})
		return
	}

	token, err := auth.GenerateTokenWithVersion(user.ID, user.Role, 0, user.TokenVersion, h.secret, h.expiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	setAuthCookie(c, token)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":               user.ID,
			"username":         user.Username,
			"email":            user.Email,
			"role":             user.Role,
			"terminal_enabled":  user.TerminalEnabled,
			"backup_enabled":    user.BackupEnabled,
			"antivirus_enabled": user.AntivirusEnabled,
			"package_id":       user.PackageID,
			"php_version":      user.PHPVersion,
			"totp_enabled":     user.TOTPEnabled,
		},
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	clearAuthCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := auth.GetUserID(c)
	user, err := h.users.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":               user.ID,
		"username":         user.Username,
		"email":            user.Email,
		"role":             user.Role,
		"terminal_enabled": user.TerminalEnabled,
		"backup_enabled":   user.BackupEnabled,
		"package_id":       user.PackageID,
		"php_version":      user.PHPVersion,
		"totp_enabled":     user.TOTPEnabled,
	})
}

// Impersonate mints a short-lived token for the target user on behalf of
// the calling admin. The token carries an ImpersonatedBy claim so audit
// logs can trace the session back to the admin who initiated it.
// Only admins may call this; the route is gated by RequireRole("admin").
func (h *AuthHandler) Impersonate(c *gin.Context) {
	targetID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	target, err := h.users.GetByID(targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if target.Status == "suspended" {
		c.JSON(http.StatusForbidden, gin.H{"error": "target account is suspended"})
		return
	}
	if target.Role == "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot impersonate admin accounts"})
		return
	}

	adminID := auth.GetUserID(c)
	token, err := auth.GenerateTokenAs(target.ID, target.Role, adminID, h.secret, "1h")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":               target.ID,
			"username":         target.Username,
			"email":            target.Email,
			"role":             target.Role,
			"terminal_enabled":  target.TerminalEnabled,
			"backup_enabled":    target.BackupEnabled,
			"antivirus_enabled": target.AntivirusEnabled,
			"package_id":       target.PackageID,
			"php_version":      target.PHPVersion,
			"totp_enabled":     target.TOTPEnabled,
		},
	})
}

// FileBrowserAuth is the auth_request endpoint nginx hits before
// proxying /filebrowser/* to the FileBrowser service.
func (h *AuthHandler) FileBrowserAuth(c *gin.Context) {
	userID := auth.GetUserID(c)
	user, err := h.users.GetByID(userID)
	if err != nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	c.Header("X-Auth-User", user.Username)
	c.Status(http.StatusOK)
}

// TOTPSetup generates a new TOTP secret and returns the QR URL + recovery
// codes. The secret is NOT yet saved — the user must confirm with a valid
// code via TOTPConfirm before 2FA is activated.
func (h *AuthHandler) TOTPSetup(c *gin.Context) {
	userID := auth.GetUserID(c)
	user, err := h.users.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "ZensPanel",
		AccountName: user.Username,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate totp: " + err.Error()})
		return
	}

	qrImage, err := key.Image(220, 220)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "generate qr: " + err.Error()})
		return
	}
	qrURL, err := qrImageDataURL(qrImage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encode qr: " + err.Error()})
		return
	}

	// Generate 8 single-use recovery codes (V29).
	recoveryCodes := make([]string, 8)
	recoveryHashes := make([]string, 8)
	for i := range recoveryCodes {
		b := make([]byte, 5)
		rand.Read(b)
		recoveryCodes[i] = fmt.Sprintf("%x", b)
		h, _ := bcrypt.GenerateFromPassword([]byte(recoveryCodes[i]), bcrypt.DefaultCost)
		recoveryHashes[i] = string(h)
	}

	// Encrypt the secret before storing (V27). Store in a pending sync.Map
	// keyed by userID — only committed on Confirm.
	enc, err := h.encryptTOTP(key.Secret())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encrypt totp: " + err.Error()})
		return
	}
	hashJSON, _ := json.Marshal(recoveryHashes)
	totpPendingStore.Store(userID, totpPending{
		secretEnc:     enc,
		recoveryCodes: string(hashJSON),
	})

	c.JSON(http.StatusOK, gin.H{
		"secret":         key.Secret(),
		"qr_url":         qrURL,
		"otpauth_url":    key.URL(),
		"recovery_codes": recoveryCodes,
	})
}

type totpPending struct {
	secretEnc     string
	recoveryCodes string
}

var totpPendingStore sync.Map // map[uint64]totpPending

// TOTPConfirm activates 2FA after the user proves they can generate a valid
// code from the secret returned by TOTPSetup.
func (h *AuthHandler) TOTPConfirm(c *gin.Context) {
	userID := auth.GetUserID(c)
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pending, ok := totpPendingStore.Load(userID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no pending 2FA setup — call /auth/2fa/setup first"})
		return
	}
	p := pending.(totpPending)

	secret, err := h.decryptTOTP(p.secretEnc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "decrypt totp"})
		return
	}
	if !totp.Validate(req.Code, secret) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid TOTP code"})
		return
	}

	if err := h.users.SetTOTP(userID, p.secretEnc, true, p.recoveryCodes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	totpPendingStore.Delete(userID)
	c.JSON(http.StatusOK, gin.H{"message": "2FA enabled"})
}

// TOTPDisable removes 2FA from the account after verifying the current code.
func (h *AuthHandler) TOTPDisable(c *gin.Context) {
	userID := auth.GetUserID(c)
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	secretEnc, enabled, err := h.users.GetTOTPSecret(userID)
	if err != nil || !enabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "2FA is not enabled"})
		return
	}
	secret, err := h.decryptTOTP(secretEnc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "decrypt totp"})
		return
	}
	if !totp.Validate(req.Code, secret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid TOTP code"})
		return
	}
	if err := h.users.SetTOTP(userID, "", false, ""); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "2FA disabled"})
}

// TOTPVerify redeems a temp_token + TOTP code and returns a full JWT (V28).
func (h *AuthHandler) TOTPVerify(c *gin.Context) {
	var req struct {
		TempToken string `json:"temp_token" binding:"required"`
		Code      string `json:"code"       binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := redeemTempToken(req.TempToken)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	secretEnc, _, err := h.users.GetTOTPSecret(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup user"})
		return
	}
	secret, err := h.decryptTOTP(secretEnc)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "decrypt totp"})
		return
	}
	if !totp.Validate(req.Code, secret) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid TOTP code"})
		return
	}

	user, err := h.users.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup user"})
		return
	}
	token, err := auth.GenerateTokenWithVersion(user.ID, user.Role, 0, user.TokenVersion, h.secret, h.expiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	setAuthCookie(c, token)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":               user.ID,
			"username":         user.Username,
			"email":            user.Email,
			"role":             user.Role,
			"terminal_enabled":  user.TerminalEnabled,
			"backup_enabled":    user.BackupEnabled,
			"antivirus_enabled": user.AntivirusEnabled,
			"package_id":       user.PackageID,
			"php_version":      user.PHPVersion,
			"totp_enabled":     user.TOTPEnabled,
		},
	})
}

// TOTPRecover redeems a temp_token + recovery code and returns a full JWT.
// The recovery code is consumed (single-use, V29) and 2FA is disabled so
// the user can re-enroll with a new device.
func (h *AuthHandler) TOTPRecover(c *gin.Context) {
	var req struct {
		TempToken    string `json:"temp_token"     binding:"required"`
		RecoveryCode string `json:"recovery_code"  binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := redeemTempToken(req.TempToken)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	matched, err := h.users.ConsumeRecoveryCode(userID, req.RecoveryCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if !matched {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid recovery code"})
		return
	}

	// Disable 2FA so the user can re-enroll with a new device.
	_ = h.users.SetTOTP(userID, "", false, "")

	user, err := h.users.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup user"})
		return
	}
	token, err := auth.GenerateTokenWithVersion(user.ID, user.Role, 0, user.TokenVersion, h.secret, h.expiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	setAuthCookie(c, token)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
		},
	})
}
