package phomemo

import (
	"image"
	"image/color"
	"testing"
)

// ── Protocol constant tests ───────────────────────────────────────────────────

func TestPrintWidthPxIs560(t *testing.T) {
	t.Parallel()
	if PrintWidthPx != 560 {
		t.Fatalf("PrintWidthPx = %d, want 560 (70mm × 8 dots/mm)", PrintWidthPx)
	}
}

func TestMediaTypeValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		got  MediaType
		want byte
	}{
		{"LabelWithGaps", MediaLabelWithGaps, 0x0A},
		{"Continuous", MediaContinuous, 0x0B},
		{"LabelWithMarks", MediaLabelWithMarks, 0x26},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if byte(tc.got) != tc.want {
				t.Fatalf("%s = 0x%02X, want 0x%02X", tc.name, byte(tc.got), tc.want)
			}
		})
	}
}

// ── header() tests ────────────────────────────────────────────────────────────

func TestHeader(t *testing.T) {
	t.Parallel()

	h := header(MediaLabelWithGaps)
	// Speed command: 1B 4E 0D 05
	if h[0] != 0x1B || h[1] != 0x4E || h[2] != 0x0D || h[3] != 0x05 {
		t.Fatalf("header speed = % x, want 1B 4E 0D 05", h[:4])
	}
	// Density command: 1B 4E 04 0A
	if h[4] != 0x1B || h[5] != 0x4E || h[6] != 0x04 || h[7] != 0x0A {
		t.Fatalf("header density = % x, want 1B 4E 04 0A", h[4:8])
	}
	// Media type: 1F 11 0A
	if h[8] != 0x1F || h[9] != 0x11 || h[10] != byte(MediaLabelWithGaps) {
		t.Fatalf("header media = % x, want 1F 11 0A", h[8:11])
	}
}

func TestHeaderContinuousMedia(t *testing.T) {
	t.Parallel()

	h := header(MediaContinuous)
	if h[10] != byte(MediaContinuous) {
		t.Fatalf("header media byte = 0x%02X, want 0x%02X", h[10], byte(MediaContinuous))
	}
}

// ── rasterCmd() tests ─────────────────────────────────────────────────────────

func TestRasterCmd(t *testing.T) {
	t.Parallel()

	cmd := rasterCmd(70, 200)
	if len(cmd) != 8 {
		t.Fatalf("rasterCmd len = %d, want 8", len(cmd))
	}
	// Prefix: GS v 0 mode
	if cmd[0] != 0x1D || cmd[1] != 0x76 || cmd[2] != 0x30 || cmd[3] != 0x00 {
		t.Fatalf("rasterCmd prefix = % x, want 1D 76 30 00", cmd[:4])
	}
	// width bytes LE: 70 = 0x46 0x00
	if cmd[4] != 0x46 || cmd[5] != 0x00 {
		t.Fatalf("rasterCmd width = % x, want 46 00", cmd[4:6])
	}
	// height LE: 200 = 0xC8 0x00
	if cmd[6] != 0xC8 || cmd[7] != 0x00 {
		t.Fatalf("rasterCmd height = % x, want C8 00", cmd[6:8])
	}
}

func TestRasterCmdHeightOver255(t *testing.T) {
	t.Parallel()

	// 300 lines = 0x012C → LE bytes: 0x2C, 0x01
	cmd := rasterCmd(1, 300)
	if cmd[6] != 0x2C || cmd[7] != 0x01 {
		t.Fatalf("rasterCmd height 300 = % x, want 2C 01", cmd[6:8])
	}
}

func TestRasterCmdMaxHeight(t *testing.T) {
	t.Parallel()

	// 0xFFFF = 65535 → LE bytes: 0xFF, 0xFF
	cmd := rasterCmd(1, 0xFFFF)
	if cmd[6] != 0xFF || cmd[7] != 0xFF {
		t.Fatalf("rasterCmd max height = % x, want FF FF", cmd[6:8])
	}
}

// ── footer() tests ────────────────────────────────────────────────────────────

func TestFooter(t *testing.T) {
	t.Parallel()

	f := footer()
	want := []byte{0x1F, 0xF0, 0x05, 0x00, 0x1F, 0xF0, 0x03, 0x00}
	if len(f) != len(want) {
		t.Fatalf("footer len = %d, want %d", len(f), len(want))
	}
	for i, b := range want {
		if f[i] != b {
			t.Fatalf("footer[%d] = 0x%02X, want 0x%02X", i, f[i], b)
		}
	}
}

// ── pack1bpp() tests ──────────────────────────────────────────────────────────

func TestPack1bpp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		img       image.Image
		width     int
		wantBytes []byte
		wantH     int
	}{
		{
			name: "all black 8x1",
			img: func() image.Image {
				img := image.NewGray(image.Rect(0, 0, 8, 1))
				for x := 0; x < 8; x++ {
					img.SetGray(x, 0, color.Gray{Y: 0})
				}
				return img
			}(),
			width:     8,
			wantBytes: []byte{0xFF},
			wantH:     1,
		},
		{
			name: "all white 8x1",
			img: func() image.Image {
				img := image.NewGray(image.Rect(0, 0, 8, 1))
				for x := 0; x < 8; x++ {
					img.SetGray(x, 0, color.Gray{Y: 255})
				}
				return img
			}(),
			width:     8,
			wantBytes: []byte{0x00},
			wantH:     1,
		},
		{
			name: "alternating 8x1",
			img: func() image.Image {
				img := image.NewGray(image.Rect(0, 0, 8, 1))
				for x := 0; x < 8; x++ {
					y := color.Gray{Y: 255}
					if x%2 == 0 {
						y = color.Gray{Y: 0}
					}
					img.SetGray(x, 0, y)
				}
				return img
			}(),
			width:     8,
			wantBytes: []byte{0xAA}, // 1010 1010
			wantH:     1,
		},
		{
			name: "16px wide 1x1 all black",
			img: func() image.Image {
				img := image.NewGray(image.Rect(0, 0, 16, 1))
				for x := 0; x < 16; x++ {
					img.SetGray(x, 0, color.Gray{Y: 0})
				}
				return img
			}(),
			width:     16,
			wantBytes: []byte{0xFF, 0xFF},
			wantH:     1,
		},
		{
			name: "2-row image both all-black",
			img: func() image.Image {
				img := image.NewGray(image.Rect(0, 0, 8, 2))
				for y := 0; y < 2; y++ {
					for x := 0; x < 8; x++ {
						img.SetGray(x, y, color.Gray{Y: 0})
					}
				}
				return img
			}(),
			width:     8,
			wantBytes: []byte{0xFF, 0xFF},
			wantH:     2,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, h := pack1bpp(tc.img, tc.width)
			if h != tc.wantH {
				t.Fatalf("pack1bpp height = %d, want %d", h, tc.wantH)
			}
			if len(got) != len(tc.wantBytes) {
				t.Fatalf("pack1bpp len = %d, want %d; bytes = % x", len(got), len(tc.wantBytes), got)
			}
			for i, b := range tc.wantBytes {
				if got[i] != b {
					t.Fatalf("pack1bpp[%d] = 0x%02X, want 0x%02X", i, got[i], b)
				}
			}
		})
	}
}

func TestPack1bppTransparentPixelsAreWhite(t *testing.T) {
	t.Parallel()

	// RGBA image with fully transparent pixel at (0,0) → must be white (0-bit)
	img := image.NewRGBA(image.Rect(0, 0, 8, 1))
	// All pixels default to zero-value: RGBA{0,0,0,0} → transparent
	got, _ := pack1bpp(img, 8)
	if got[0] != 0x00 {
		t.Fatalf("transparent pixel = 0x%02X, want 0x00 (white/no-print)", got[0])
	}
}

func TestPack1bppMidGrayIsWhite(t *testing.T) {
	t.Parallel()

	// Gray value 128 → lum ≈ 32896 which is just above the 0x8000 threshold → white
	img := image.NewGray(image.Rect(0, 0, 8, 1))
	for x := 0; x < 8; x++ {
		img.SetGray(x, 0, color.Gray{Y: 128})
	}
	got, _ := pack1bpp(img, 8)
	if got[0] != 0x00 {
		t.Fatalf("mid-gray (128) = 0x%02X, want 0x00 (white)", got[0])
	}
}

func TestPack1bppJustBelowThresholdIsBlack(t *testing.T) {
	t.Parallel()

	// Gray value 127 → lum < 0x8000 → black
	img := image.NewGray(image.Rect(0, 0, 8, 1))
	for x := 0; x < 8; x++ {
		img.SetGray(x, 0, color.Gray{Y: 127})
	}
	got, _ := pack1bpp(img, 8)
	if got[0] != 0xFF {
		t.Fatalf("gray(127) = 0x%02X, want 0xFF (black)", got[0])
	}
}

// ── RenderGray() tests ────────────────────────────────────────────────────────

func TestRenderGray(t *testing.T) {
	t.Parallel()

	src := image.NewGray(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			src.SetGray(x, y, color.Gray{Y: uint8(x * 64)})
		}
	}

	got := RenderGray(src)
	if got.Bounds() != src.Bounds() {
		t.Fatalf("RenderGray bounds = %v, want %v", got.Bounds(), src.Bounds())
	}
	// Spot-check a pixel: (1,0) should be ~gray 64.
	g := got.GrayAt(1, 0).Y
	if g < 60 || g > 70 {
		t.Fatalf("RenderGray pixel (1,0) = %d, want ~64", g)
	}
}
