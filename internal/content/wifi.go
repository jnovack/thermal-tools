// Package content provides image generators for common print content types.
package content

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// WifiAuth is the WiFi security type for QR code generation.
type WifiAuth string

const (
	WifiWPA    WifiAuth = "WPA"
	WifiWEP    WifiAuth = "WEP"
	WifiNoPass WifiAuth = "nopass"
)

// WifiQR generates a QR code image encoding WiFi credentials.
// The image is sized to fit within size×size pixels.
// hidden=true marks the network as a hidden SSID.
func WifiQR(ssid, password string, auth WifiAuth, hidden bool, size int) (image.Image, error) {
	if auth == "" {
		auth = WifiWPA
	}
	var sb strings.Builder
	sb.WriteString("WIFI:S:")
	sb.WriteString(escapeWifi(ssid))
	sb.WriteString(";T:")
	sb.WriteString(string(auth))
	sb.WriteString(";P:")
	sb.WriteString(escapeWifi(password))
	sb.WriteString(";H:")
	if hidden {
		sb.WriteString("true")
	} else {
		sb.WriteString("false")
	}
	sb.WriteString(";;")

	qr, err := qrcode.New(sb.String(), qrcode.Medium)
	if err != nil {
		return nil, fmt.Errorf("generate wifi QR code: %w", err)
	}
	qr.DisableBorder = false
	img := qr.Image(size)
	return toGrayImage(img), nil
}

// escapeWifi escapes special characters in SSID/password per the WiFi QR spec.
func escapeWifi(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\', ';', ',', '"', ':':
			b.WriteRune('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// toGrayImage converts an image to grayscale.
func toGrayImage(src image.Image) *image.Gray {
	b := src.Bounds()
	dst := image.NewGray(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(x, y, color.GrayModel.Convert(src.At(x, y)))
		}
	}
	return dst
}
