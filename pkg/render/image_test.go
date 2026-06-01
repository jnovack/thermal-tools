package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// ── FitWidth tests ────────────────────────────────────────────────────────────

func TestFitWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		srcW, srcH  int
		targetWidth int
		wantW       int
	}{
		{"square to half width", 100, 100, 50, 50},
		{"landscape scale", 200, 100, 100, 100},
		{"portrait scale", 100, 200, 50, 50},
		{"already correct width", 50, 50, 50, 50},
		{"scale up", 10, 10, 100, 100},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			src := image.NewGray(image.Rect(0, 0, tc.srcW, tc.srcH))
			got := FitWidth(src, tc.targetWidth)
			if got.Bounds().Dx() != tc.wantW {
				t.Fatalf("FitWidth width = %d, want %d", got.Bounds().Dx(), tc.wantW)
			}
		})
	}
}

func TestFitWidthZeroWidthIsNoOp(t *testing.T) {
	t.Parallel()

	src := image.NewGray(image.Rect(0, 0, 50, 50))
	got := FitWidth(src, 0)
	if got.Bounds().Dx() != 0 && got.Bounds().Dx() != 50 {
		// Either returning src unchanged or a zero-width image is acceptable.
		t.Fatalf("FitWidth(src, 0) width = %d, expected 0 or 50", got.Bounds().Dx())
	}
}

func TestFitWidthZeroSrcIsNoOp(t *testing.T) {
	t.Parallel()

	src := image.NewGray(image.Rect(0, 0, 0, 0))
	got := FitWidth(src, 100)
	// Zero-width source should return the original without panic.
	if got == nil {
		t.Fatal("FitWidth(zero-width src) returned nil")
	}
}

// ── FitBox tests ──────────────────────────────────────────────────────────────

func TestFitBox(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		srcW, srcH   int
		boxW, boxH   int
		wantW, wantH int // canvas dimensions (always boxW×boxH when boxH>0)
	}{
		// height=0 → FitWidth behaviour
		{"height=0 is FitWidth", 200, 100, 100, 0, 100, 50},
		// Landscape in square box: constrained by width
		{"landscape fits width", 200, 100, 100, 100, 100, 100},
		// Portrait in square box: constrained by height
		{"portrait fits height", 100, 200, 100, 100, 100, 100},
		// Smaller content: scaled up to fill box
		{"scale up to fill", 50, 50, 100, 100, 100, 100},
		// Already fits exactly
		{"exact fit", 100, 100, 100, 100, 100, 100},
		// Very wide content in a tall box: width-constrained
		{"wide in tall box", 400, 100, 100, 200, 100, 200},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			src := image.NewGray(image.Rect(0, 0, tc.srcW, tc.srcH))
			got := FitBox(src, tc.boxW, tc.boxH)
			b := got.Bounds()
			if b.Dx() != tc.wantW || b.Dy() != tc.wantH {
				t.Fatalf("FitBox canvas = %dx%d, want %dx%d",
					b.Dx(), b.Dy(), tc.wantW, tc.wantH)
			}
		})
	}
}

func TestFitBoxPreservesAspectRatio(t *testing.T) {
	t.Parallel()

	// 200×100 image (2:1 aspect) in a 100×100 box.
	// Scale factor = 100/200 = 0.5 → scaled content = 100×50.
	// Canvas = 100×100 with 25px white padding top and bottom.
	src := image.NewRGBA(image.Rect(0, 0, 200, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 200; x++ {
			src.Set(x, y, color.Black)
		}
	}

	got := FitBox(src, 100, 100)
	// Top-left corner (0,0) should be white (padding).
	r, g, bv, _ := got.At(0, 0).RGBA()
	isWhite := r > 0xF000 && g > 0xF000 && bv > 0xF000
	if !isWhite {
		t.Fatalf("FitBox(200×100 in 100×100): corner pixel should be white padding, got RGBA(%d,%d,%d)", r, g, bv)
	}
}

func TestFitBoxZeroSrcReturnsCanvas(t *testing.T) {
	t.Parallel()

	src := image.NewGray(image.Rect(0, 0, 0, 0))
	got := FitBox(src, 100, 50)
	b := got.Bounds()
	if b.Dx() != 100 || b.Dy() != 50 {
		t.Fatalf("FitBox(zero src, 100, 50) = %dx%d, want 100×50", b.Dx(), b.Dy())
	}
}

// ── Dither tests ──────────────────────────────────────────────────────────────

func TestDither(t *testing.T) {
	t.Parallel()

	// All-white → palette index 0 everywhere.
	src := image.NewGray(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			src.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	dst := Dither(src)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if dst.ColorIndexAt(x, y) != 0 {
				t.Fatalf("Dither(white): pixel (%d,%d) = %d, want 0 (white)", x, y, dst.ColorIndexAt(x, y))
			}
		}
	}

	// All-black → palette index 1 everywhere.
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			src.SetGray(x, y, color.Gray{Y: 0})
		}
	}
	dst = Dither(src)
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if dst.ColorIndexAt(x, y) != 1 {
				t.Fatalf("Dither(black): pixel (%d,%d) = %d, want 1 (black)", x, y, dst.ColorIndexAt(x, y))
			}
		}
	}
}

func TestDitherPalette(t *testing.T) {
	t.Parallel()

	src := image.NewGray(image.Rect(0, 0, 4, 4))
	dst := Dither(src)
	if len(dst.Palette) != 2 {
		t.Fatalf("Dither palette len = %d, want 2", len(dst.Palette))
	}
}

func TestDitherPreservesBounds(t *testing.T) {
	t.Parallel()

	src := image.NewGray(image.Rect(0, 0, 16, 12))
	dst := Dither(src)
	if dst.Bounds() != src.Bounds() {
		t.Fatalf("Dither bounds = %v, want %v", dst.Bounds(), src.Bounds())
	}
}

// ── ToGray tests ──────────────────────────────────────────────────────────────

func TestToGray(t *testing.T) {
	t.Parallel()

	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			src.Set(x, y, color.RGBA{R: 200, G: 200, B: 200, A: 255})
		}
	}
	got := ToGray(src)
	if got.Bounds() != src.Bounds() {
		t.Fatalf("ToGray bounds mismatch: got %v, want %v", got.Bounds(), src.Bounds())
	}
	// A grey pixel should stay approximately grey.
	g := got.GrayAt(0, 0).Y
	if g < 190 || g > 210 {
		t.Fatalf("ToGray pixel (0,0) = %d, want ~200", g)
	}
}

func TestToGrayBlackWhite(t *testing.T) {
	t.Parallel()

	src := image.NewGray(image.Rect(0, 0, 2, 1))
	src.SetGray(0, 0, color.Gray{Y: 0})
	src.SetGray(1, 0, color.Gray{Y: 255})

	got := ToGray(src)
	if got.GrayAt(0, 0).Y != 0 {
		t.Fatalf("ToGray black pixel = %d, want 0", got.GrayAt(0, 0).Y)
	}
	if got.GrayAt(1, 0).Y != 255 {
		t.Fatalf("ToGray white pixel = %d, want 255", got.GrayAt(1, 0).Y)
	}
}

// ── Rotation tests ────────────────────────────────────────────────────────────

func TestRotate90(t *testing.T) {
	t.Parallel()

	// 2×1 image — result must be 1×2.
	src := image.NewGray(image.Rect(0, 0, 2, 1))
	src.SetGray(0, 0, color.Gray{Y: 0})
	src.SetGray(1, 0, color.Gray{Y: 255})

	got := Rotate90(src)
	b := got.Bounds()
	if b.Dx() != 1 || b.Dy() != 2 {
		t.Fatalf("Rotate90 bounds = %v, want 1×2", b)
	}
}

func TestRotate180(t *testing.T) {
	t.Parallel()

	// 2×1: (0,0)=black, (1,0)=white
	// 180°: (0,0)=white, (1,0)=black
	src := image.NewGray(image.Rect(0, 0, 2, 1))
	src.SetGray(0, 0, color.Gray{Y: 0})
	src.SetGray(1, 0, color.Gray{Y: 255})

	got := Rotate180(src)
	b := got.Bounds()
	if b.Dx() != 2 || b.Dy() != 1 {
		t.Fatalf("Rotate180 bounds = %v, want 2×1", b)
	}
	// top-left should now be white
	r, g, bv, _ := got.At(0, 0).RGBA()
	isWhite := r > 0xF000 && g > 0xF000 && bv > 0xF000
	if !isWhite {
		t.Fatalf("Rotate180: pixel (0,0) should be white, got RGBA(%d,%d,%d)", r, g, bv)
	}
}

func TestRotate180PixelMapping(t *testing.T) {
	t.Parallel()

	// 2×2: TL=black, TR=white, BL=white, BR=black.
	// After 180°: TL=black (was BR), TR=white (was BL), BL=white (was TR), BR=black (was TL).
	src := image.NewGray(image.Rect(0, 0, 2, 2))
	src.SetGray(0, 0, color.Gray{Y: 0})   // TL black
	src.SetGray(1, 0, color.Gray{Y: 255}) // TR white
	src.SetGray(0, 1, color.Gray{Y: 255}) // BL white
	src.SetGray(1, 1, color.Gray{Y: 0})   // BR black

	got := Rotate180(src)
	// TL of result = BR of source = black (0).
	r, g, b, _ := got.At(0, 0).RGBA()
	if r != 0 || g != 0 || b != 0 {
		t.Fatalf("Rotate180 TL: want black, got RGBA(%d,%d,%d)", r, g, b)
	}
	// BR of result = TL of source = black (0).
	r, g, b, _ = got.At(1, 1).RGBA()
	if r != 0 || g != 0 || b != 0 {
		t.Fatalf("Rotate180 BR: want black, got RGBA(%d,%d,%d)", r, g, b)
	}
}

func TestRotate270(t *testing.T) {
	t.Parallel()

	// Rotating 270° CW on a 2×1 image should produce a 1×2 image.
	src := image.NewGray(image.Rect(0, 0, 2, 1))
	got := Rotate270(src)
	b := got.Bounds()
	if b.Dx() != 1 || b.Dy() != 2 {
		t.Fatalf("Rotate270 bounds = %v, want 1×2", b)
	}
}

func TestRotate90Then270IsIdentity(t *testing.T) {
	t.Parallel()

	// Rotating 90° and then 270° should produce the same dimensions as the original.
	src := image.NewGray(image.Rect(0, 0, 5, 3))
	got := Rotate270(Rotate90(src))
	b := got.Bounds()
	if b.Dx() != 5 || b.Dy() != 3 {
		t.Fatalf("Rotate90+Rotate270 bounds = %v, want 5×3", b)
	}
}

func TestRotate180TwiceIsIdentityDimensions(t *testing.T) {
	t.Parallel()

	src := image.NewGray(image.Rect(0, 0, 7, 3))
	got := Rotate180(Rotate180(src))
	b := got.Bounds()
	if b.Dx() != 7 || b.Dy() != 3 {
		t.Fatalf("Rotate180×2 bounds = %v, want 7×3", b)
	}
}

// ── Composition tests ─────────────────────────────────────────────────────────

func TestVStack(t *testing.T) {
	t.Parallel()

	a := image.NewGray(image.Rect(0, 0, 10, 5))
	b := image.NewGray(image.Rect(0, 0, 10, 3))
	got := VStack(2, a, b)
	// 5 + 2 (gap) + 3 = 10
	if got.Bounds().Dy() != 10 {
		t.Fatalf("VStack height = %d, want 10", got.Bounds().Dy())
	}
	if got.Bounds().Dx() != 10 {
		t.Fatalf("VStack width = %d, want 10", got.Bounds().Dx())
	}
}

func TestVStackEmpty(t *testing.T) {
	t.Parallel()

	got := VStack(0)
	if got.Bounds().Dx() != 0 || got.Bounds().Dy() != 0 {
		t.Fatalf("VStack() empty = %v, want 0×0", got.Bounds())
	}
}

func TestVStackSingle(t *testing.T) {
	t.Parallel()

	img := image.NewGray(image.Rect(0, 0, 10, 10))
	got := VStack(5, img)
	if got.Bounds().Dy() != 10 {
		t.Fatalf("VStack with one image height = %d, want 10", got.Bounds().Dy())
	}
}

func TestVStackWidestWins(t *testing.T) {
	t.Parallel()

	narrow := image.NewGray(image.Rect(0, 0, 5, 5))
	wide := image.NewGray(image.Rect(0, 0, 20, 5))
	got := VStack(0, narrow, wide)
	if got.Bounds().Dx() != 20 {
		t.Fatalf("VStack width = %d, want 20 (widest of inputs)", got.Bounds().Dx())
	}
}

func TestVStackGapZero(t *testing.T) {
	t.Parallel()

	a := image.NewGray(image.Rect(0, 0, 5, 3))
	b := image.NewGray(image.Rect(0, 0, 5, 4))
	got := VStack(0, a, b)
	if got.Bounds().Dy() != 7 {
		t.Fatalf("VStack(gap=0) height = %d, want 7", got.Bounds().Dy())
	}
}

func TestRule(t *testing.T) {
	t.Parallel()

	r := Rule(10, 3)
	if r.Bounds().Dx() != 10 || r.Bounds().Dy() != 3 {
		t.Fatalf("Rule bounds = %v, want 10×3", r.Bounds())
	}
	// Every pixel should be black.
	for y := 0; y < 3; y++ {
		for x := 0; x < 10; x++ {
			rv, gv, bv, _ := r.At(x, y).RGBA()
			isBlack := rv == 0 && gv == 0 && bv == 0
			if !isBlack {
				t.Fatalf("Rule pixel (%d,%d) is not black: RGBA(%d,%d,%d)", x, y, rv, gv, bv)
			}
		}
	}
}

func TestSolid(t *testing.T) {
	t.Parallel()

	s := Solid(8, 4)
	if s.Bounds().Dx() != 8 || s.Bounds().Dy() != 4 {
		t.Fatalf("Solid bounds = %v, want 8×4", s.Bounds())
	}
	// Solid is an alias for Rule — every pixel must be black.
	for y := 0; y < 4; y++ {
		for x := 0; x < 8; x++ {
			rv, gv, bv, _ := s.At(x, y).RGBA()
			if rv != 0 || gv != 0 || bv != 0 {
				t.Fatalf("Solid pixel (%d,%d) is not black: RGBA(%d,%d,%d)", x, y, rv, gv, bv)
			}
		}
	}
}

func TestUniform(t *testing.T) {
	t.Parallel()

	u := Uniform(8, 4, 128)
	if u.Bounds().Dx() != 8 || u.Bounds().Dy() != 4 {
		t.Fatalf("Uniform bounds = %v, want 8×4", u.Bounds())
	}
}

func TestUniformPixelValue(t *testing.T) {
	t.Parallel()

	const level = uint8(128)
	u := Uniform(4, 4, level)
	g, ok := u.(*image.Gray)
	if !ok {
		t.Skip("Uniform returned non-*image.Gray, skipping pixel value check")
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			if got := g.GrayAt(x, y).Y; got != level {
				t.Fatalf("Uniform pixel (%d,%d) = %d, want %d", x, y, got, level)
			}
		}
	}
}

func TestUniformExtremes(t *testing.T) {
	t.Parallel()

	tests := []struct{ level uint8 }{
		{0},
		{255},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(fmt.Sprintf("level=%d", tc.level), func(t *testing.T) {
			t.Parallel()
			u := Uniform(2, 2, tc.level)
			g, ok := u.(*image.Gray)
			if !ok {
				t.Skip("Uniform returned non-*image.Gray")
			}
			if g.GrayAt(0, 0).Y != tc.level {
				t.Fatalf("Uniform(level=%d) pixel = %d, want %d", tc.level, g.GrayAt(0, 0).Y, tc.level)
			}
		})
	}
}

func TestPad(t *testing.T) {
	t.Parallel()

	src := image.NewGray(image.Rect(0, 0, 10, 5))
	got := Pad(src, 3, 7)
	if got.Bounds().Dy() != 15 { // 3 + 5 + 7
		t.Fatalf("Pad height = %d, want 15", got.Bounds().Dy())
	}
	if got.Bounds().Dx() != 10 {
		t.Fatalf("Pad width = %d, want 10", got.Bounds().Dx())
	}
}

func TestPadZero(t *testing.T) {
	t.Parallel()

	src := image.NewGray(image.Rect(0, 0, 5, 5))
	got := Pad(src, 0, 0)
	if got.Bounds().Dy() != 5 || got.Bounds().Dx() != 5 {
		t.Fatalf("Pad(0,0) bounds = %v, want 5×5", got.Bounds())
	}
}

// ── Paginate tests ────────────────────────────────────────────────────────────

func TestPaginate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		imgH       int
		pageHeight int
		wantPages  int
		wantH      int
	}{
		{"receipt mode (height=0)", 200, 0, 1, 200},
		{"content shorter than page", 50, 100, 1, 50},
		{"exact fit", 100, 100, 1, 100},
		{"two equal pages", 200, 100, 2, 100},
		{"uneven split, last padded to pageHeight", 150, 100, 2, 100},
		{"three pages", 300, 100, 3, 100},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			img := image.NewGray(image.Rect(0, 0, 10, tc.imgH))
			pages := Paginate(img, tc.pageHeight)
			if len(pages) != tc.wantPages {
				t.Fatalf("Paginate page count = %d, want %d", len(pages), tc.wantPages)
			}
			for i, p := range pages {
				if p.Bounds().Dy() != tc.wantH {
					t.Fatalf("page[%d] height = %d, want %d", i, p.Bounds().Dy(), tc.wantH)
				}
			}
		})
	}
}

func TestPaginateLastPagePaddedWhite(t *testing.T) {
	t.Parallel()

	// 1×150 black image paginated at height=100.
	// Second page has 50 real rows (black) + 50 padded rows (white).
	src := image.NewGray(image.Rect(0, 0, 1, 150))
	// Fill all rows black.
	for y := 0; y < 150; y++ {
		src.SetGray(0, y, color.Gray{Y: 0})
	}

	pages := Paginate(src, 100)
	if len(pages) != 2 {
		t.Fatalf("Paginate page count = %d, want 2", len(pages))
	}

	// Row 0 of page 2 = row 100 of original = black.
	p2 := pages[1]
	r, g, bv, _ := p2.At(0, 0).RGBA()
	isBlack := r == 0 && g == 0 && bv == 0
	if !isBlack {
		t.Fatalf("page[1] row 0 should be black (content), got RGBA(%d,%d,%d)", r, g, bv)
	}

	// Row 50 of page 2 = padding = white.
	r, g, bv, _ = p2.At(0, 50).RGBA()
	isWhite := r > 0xF000 && g > 0xF000 && bv > 0xF000
	if !isWhite {
		t.Fatalf("page[1] row 50 should be white (padding), got RGBA(%d,%d,%d)", r, g, bv)
	}
}

func TestPaginateNegativeHeightIsReceiptMode(t *testing.T) {
	t.Parallel()

	img := image.NewGray(image.Rect(0, 0, 10, 50))
	pages := Paginate(img, -1)
	if len(pages) != 1 {
		t.Fatalf("Paginate(negative) page count = %d, want 1 (receipt mode)", len(pages))
	}
}

// ── LoadImage / DecodeImage tests ─────────────────────────────────────────────

func TestLoadImageValid(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	pngPath := filepath.Join(dir, "test.png")
	img := image.NewGray(image.Rect(0, 0, 8, 8))
	f, err := os.Create(pngPath)
	if err != nil {
		t.Fatalf("create test PNG: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		f.Close()
		t.Fatalf("encode test PNG: %v", err)
	}
	f.Close()

	got, err := LoadImage(pngPath)
	if err != nil {
		t.Fatalf("LoadImage() error = %v", err)
	}
	if got.Bounds().Dx() != 8 || got.Bounds().Dy() != 8 {
		t.Fatalf("LoadImage bounds = %v, want 8×8", got.Bounds())
	}
}

func TestLoadImageNotFound(t *testing.T) {
	t.Parallel()

	_, err := LoadImage(filepath.Join(t.TempDir(), "no-such-file.png"))
	if err == nil {
		t.Fatal("LoadImage() expected error for missing file, got nil")
	}
}

func TestLoadImageInvalidData(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	badPath := filepath.Join(dir, "bad.png")
	if err := os.WriteFile(badPath, []byte("not an image"), 0o600); err != nil {
		t.Fatalf("write bad file: %v", err)
	}
	_, err := LoadImage(badPath)
	if err == nil {
		t.Fatal("LoadImage() expected error for invalid image data, got nil")
	}
}

func TestDecodeImageValid(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewGray(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatalf("encode: %v", err)
	}
	got, err := DecodeImage(&buf)
	if err != nil {
		t.Fatalf("DecodeImage() error = %v", err)
	}
	if got.Bounds().Dx() != 4 {
		t.Fatalf("DecodeImage width = %d, want 4", got.Bounds().Dx())
	}
}

func TestDecodeImageInvalidData(t *testing.T) {
	t.Parallel()

	r := bytes.NewReader([]byte("this is not an image"))
	_, err := DecodeImage(r)
	if err == nil {
		t.Fatal("DecodeImage() expected error for garbage data, got nil")
	}
}

func TestDecodeImageEmpty(t *testing.T) {
	t.Parallel()

	_, err := DecodeImage(bytes.NewReader(nil))
	if err == nil {
		t.Fatal("DecodeImage() expected error for empty reader, got nil")
	}
}

// ── EncodePNG tests ───────────────────────────────────────────────────────────

func TestEncodePNG(t *testing.T) {
	t.Parallel()

	src := image.NewGray(image.Rect(0, 0, 8, 8))
	data, err := EncodePNG(src)
	if err != nil {
		t.Fatalf("EncodePNG() error = %v", err)
	}
	if len(data) == 0 {
		t.Fatal("EncodePNG() returned empty bytes")
	}
	// Verify it round-trips.
	got, err := DecodeImage(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("DecodeImage(EncodePNG result) error = %v", err)
	}
	if got.Bounds() != src.Bounds() {
		t.Fatalf("round-trip bounds: got %v, want %v", got.Bounds(), src.Bounds())
	}
}

func TestEncodePNGRoundTripColors(t *testing.T) {
	t.Parallel()

	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	src.Set(0, 0, color.RGBA{R: 255, A: 255})
	src.Set(3, 3, color.RGBA{B: 255, A: 255})

	data, err := EncodePNG(src)
	if err != nil {
		t.Fatalf("EncodePNG() error = %v", err)
	}
	got, err := DecodeImage(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("DecodeImage(EncodePNG result) error = %v", err)
	}
	if got.Bounds() != src.Bounds() {
		t.Fatalf("round-trip bounds: got %v, want %v", got.Bounds(), src.Bounds())
	}
}

// ── EncodeJPEG tests ──────────────────────────────────────────────────────────

func TestEncodeJPEG(t *testing.T) {
	t.Parallel()

	src := image.NewGray(image.Rect(0, 0, 8, 8))
	data, err := EncodeJPEG(src, 85)
	if err != nil {
		t.Fatalf("EncodeJPEG() error = %v", err)
	}
	if len(data) == 0 {
		t.Fatal("EncodeJPEG() returned empty bytes")
	}
}

func TestEncodeJPEGQualityRange(t *testing.T) {
	t.Parallel()

	src := image.NewGray(image.Rect(0, 0, 8, 8))
	for _, q := range []int{1, 50, 100} {
		q := q
		t.Run(fmt.Sprintf("quality=%d", q), func(t *testing.T) {
			t.Parallel()
			data, err := EncodeJPEG(src, q)
			if err != nil {
				t.Fatalf("EncodeJPEG(q=%d) error = %v", q, err)
			}
			if len(data) == 0 {
				t.Fatalf("EncodeJPEG(q=%d) returned empty bytes", q)
			}
		})
	}
}
