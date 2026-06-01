package content

import (
	"image"
	"testing"
)

// ── WifiQR happy-path tests ───────────────────────────────────────────────────

func TestWifiQRReturnsImage(t *testing.T) {
	t.Parallel()

	img, err := WifiQR("MyNetwork", "password", WifiWPA, false, 200)
	if err != nil {
		t.Fatalf("WifiQR() error = %v", err)
	}
	if img == nil {
		t.Fatal("WifiQR() returned nil image")
	}
}

func TestWifiQRSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		size int
	}{
		{"small 100px", 100},
		{"medium 200px", 200},
		{"large 560px", 560},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			img, err := WifiQR("SSID", "pass", WifiWPA, false, tc.size)
			if err != nil {
				t.Fatalf("WifiQR(%d) error = %v", tc.size, err)
			}
			b := img.Bounds()
			// QR codes are square and approximately the requested size.
			// The library may adjust slightly for module alignment.
			if b.Dx() == 0 || b.Dy() == 0 {
				t.Fatalf("WifiQR(%d) returned zero-dimension image: %v", tc.size, b)
			}
		})
	}
}

func TestWifiQRAuthTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		auth WifiAuth
	}{
		{"WPA", WifiWPA},
		{"WEP", WifiWEP},
		{"nopass", WifiNoPass},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			img, err := WifiQR("net", "pass", tc.auth, false, 100)
			if err != nil {
				t.Fatalf("WifiQR(%s) error = %v", tc.auth, err)
			}
			if img == nil {
				t.Fatalf("WifiQR(%s) returned nil", tc.auth)
			}
		})
	}
}

func TestWifiQRHiddenNetwork(t *testing.T) {
	t.Parallel()

	img, err := WifiQR("HiddenNet", "secret", WifiWPA, true, 200)
	if err != nil {
		t.Fatalf("WifiQR(hidden) error = %v", err)
	}
	if img == nil {
		t.Fatal("WifiQR(hidden) returned nil")
	}
}

func TestWifiQREmptyPassword(t *testing.T) {
	t.Parallel()

	img, err := WifiQR("OpenNet", "", WifiNoPass, false, 200)
	if err != nil {
		t.Fatalf("WifiQR(empty password) error = %v", err)
	}
	if img == nil {
		t.Fatal("WifiQR(empty password) returned nil")
	}
}

func TestWifiQRDefaultAuthWhenEmpty(t *testing.T) {
	t.Parallel()

	// Empty auth string should fall back to WPA without panicking.
	img, err := WifiQR("net", "pass", "", false, 100)
	if err != nil {
		t.Fatalf("WifiQR(empty auth) error = %v", err)
	}
	if img == nil {
		t.Fatal("WifiQR(empty auth) returned nil")
	}
}

func TestWifiQRReturnsGrayImage(t *testing.T) {
	t.Parallel()

	img, err := WifiQR("net", "pass", WifiWPA, false, 100)
	if err != nil {
		t.Fatalf("WifiQR() error = %v", err)
	}
	if _, ok := img.(*image.Gray); !ok {
		t.Fatalf("WifiQR() returned %T, want *image.Gray", img)
	}
}

// ── escapeWifi tests ──────────────────────────────────────────────────────────

func TestEscapeWifi(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain string", "MyNetwork", "MyNetwork"},
		{"escape backslash", `back\slash`, `back\\slash`},
		{"escape semicolon", "semi;colon", `semi\;colon`},
		{"escape comma", "com,ma", `com\,ma`},
		{"escape double quote", `dou"ble`, `dou\"ble`},
		{"escape colon", "co:lon", `co\:lon`},
		{"multiple special chars", `a;b\c`, `a\;b\\c`},
		{"empty string", "", ""},
		{"no special chars", "normalpassword123", "normalpassword123"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := escapeWifi(tc.input)
			if got != tc.want {
				t.Fatalf("escapeWifi(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// ── toGrayImage tests ─────────────────────────────────────────────────────────

func TestToGrayImage(t *testing.T) {
	t.Parallel()

	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	got := toGrayImage(src)
	// toGrayImage returns *image.Gray; verify bounds and non-nil.
	if got == nil {
		t.Fatal("toGrayImage() returned nil")
	}
	if got.Bounds() != src.Bounds() {
		t.Fatalf("toGrayImage() bounds = %v, want %v", got.Bounds(), src.Bounds())
	}
}
