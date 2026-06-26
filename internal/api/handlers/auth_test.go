package handlers

import (
	"image"
	"strings"
	"testing"
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
