package handlers

import (
	"image"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestV69_QRImageDataURLPNG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))

	dataURL, err := qrImageDataURL(img)
	if err != nil {
		t.Fatalf("qrImageDataURL returned error: %v", err)
	}

	if !strings.HasPrefix(dataURL, "data:image/png;base64,") {
		t.Fatalf("qrImageDataURL prefix = %q", dataURL[:min(len(dataURL), 32)])
	}
}

func TestV72_LogoutClearsAuthCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rec := httptest.NewRecorder()

	h := NewAuthHandler(nil, "secret", "24h", "")
	c, _ := gin.CreateTestContext(rec)
	c.Request = req
	h.Logout(c)

	setCookie := rec.Header().Get("Set-Cookie")
	for _, want := range []string{
		"zenspanel_token=",
		"Max-Age=0",
		"HttpOnly",
		"Secure",
		"SameSite=Strict",
	} {
		if !strings.Contains(setCookie, want) {
			t.Fatalf("Set-Cookie = %q, missing %q", setCookie, want)
		}
	}
}
