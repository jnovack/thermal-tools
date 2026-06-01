package render

import (
	"image/color"
	"testing"

	"golang.org/x/image/font/basicfont"
)

// ── Text tests ────────────────────────────────────────────────────────────────

func TestTextNilConfigNoPanic(t *testing.T) {
	t.Parallel()

	img := Text("hello", 200, nil)
	if img == nil {
		t.Fatal("Text() returned nil with nil config")
	}
	if img.Bounds().Dx() != 200 {
		t.Fatalf("Text() width = %d, want 200", img.Bounds().Dx())
	}
	if img.Bounds().Dy() <= 0 {
		t.Fatal("Text() produced zero-height image")
	}
}

func TestTextEmpty(t *testing.T) {
	t.Parallel()

	img := Text("", 100, nil)
	if img == nil {
		t.Fatal("Text('') returned nil")
	}
	if img.Bounds().Dy() <= 0 {
		t.Fatal("Text('') produced zero-height image")
	}
}

func TestTextSingleLine(t *testing.T) {
	t.Parallel()

	img := Text("hello world", 300, nil)
	if img == nil {
		t.Fatal("Text() returned nil")
	}
	if img.Bounds().Dx() != 300 {
		t.Fatalf("Text() width = %d, want 300", img.Bounds().Dx())
	}
}

func TestTextWordWrapProducesTallerImage(t *testing.T) {
	t.Parallel()

	// Render the same words into a wide canvas (single line) and a narrow one
	// (forces wrapping). The narrow result must be taller.
	wide := Text("the quick brown fox", 400, nil)
	narrow := Text("the quick brown fox", 30, nil)
	if narrow.Bounds().Dy() <= wide.Bounds().Dy() {
		t.Fatalf("word-wrap: narrow height (%d) should exceed wide height (%d)",
			narrow.Bounds().Dy(), wide.Bounds().Dy())
	}
}

func TestTextExplicitNewlinesProduceTallerImage(t *testing.T) {
	t.Parallel()

	single := Text("line", 200, nil)
	multi := Text("line\nline\nline", 200, nil)
	if multi.Bounds().Dy() <= single.Bounds().Dy() {
		t.Fatalf("multi-line height (%d) should exceed single-line height (%d)",
			multi.Bounds().Dy(), single.Bounds().Dy())
	}
}

func TestTextVeryLongWordNoBreakPoint(t *testing.T) {
	t.Parallel()

	// A single word longer than the canvas width must not panic.
	img := Text("supercalifragilisticexpialidocious", 10, nil)
	if img == nil {
		t.Fatal("Text() returned nil for very long word")
	}
	if img.Bounds().Dy() <= 0 {
		t.Fatal("Text() produced zero-height image for very long word")
	}
}

func TestTextCustomConfig(t *testing.T) {
	t.Parallel()

	cfg := &TextConfig{
		Face:        basicfont.Face7x13,
		LineSpacing: 5,
		Background:  color.Black,
		Foreground:  color.White,
	}
	img := Text("hello", 200, cfg)
	if img == nil {
		t.Fatal("Text() with custom config returned nil")
	}
	if img.Bounds().Dx() != 200 {
		t.Fatalf("Text() with custom config width = %d, want 200", img.Bounds().Dx())
	}
}

func TestTextLineSpacingAffectsHeight(t *testing.T) {
	t.Parallel()

	narrow := Text("a\nb", 200, &TextConfig{LineSpacing: 1})
	wide := Text("a\nb", 200, &TextConfig{LineSpacing: 20})
	if wide.Bounds().Dy() <= narrow.Bounds().Dy() {
		t.Fatalf("larger line spacing height (%d) should exceed smaller spacing height (%d)",
			wide.Bounds().Dy(), narrow.Bounds().Dy())
	}
}

func TestTextOnlyNewlines(t *testing.T) {
	t.Parallel()

	// Multiple newlines with no text — should still produce a valid image.
	img := Text("\n\n\n", 100, nil)
	if img == nil {
		t.Fatal("Text(only newlines) returned nil")
	}
	if img.Bounds().Dy() <= 0 {
		t.Fatal("Text(only newlines) produced zero-height image")
	}
}

// ── TextConfig method tests ───────────────────────────────────────────────────

func TestTextConfigNilReceiver(t *testing.T) {
	t.Parallel()

	var cfg *TextConfig

	if f := cfg.face(); f == nil {
		t.Fatal("nil TextConfig.face() returned nil")
	}
	if s := cfg.lineSpacing(); s <= 0 {
		t.Fatalf("nil TextConfig.lineSpacing() = %d, want > 0", s)
	}
	if b := cfg.bg(); b == nil {
		t.Fatal("nil TextConfig.bg() returned nil")
	}
	if f := cfg.fg(); f == nil {
		t.Fatal("nil TextConfig.fg() returned nil")
	}
}

func TestTextConfigEmptyStruct(t *testing.T) {
	t.Parallel()

	cfg := &TextConfig{} // non-nil but all zero values

	if f := cfg.face(); f == nil {
		t.Fatal("empty TextConfig.face() returned nil")
	}
	if s := cfg.lineSpacing(); s <= 0 {
		t.Fatalf("empty TextConfig.lineSpacing() = %d, want > 0", s)
	}
	if b := cfg.bg(); b == nil {
		t.Fatal("empty TextConfig.bg() returned nil")
	}
	if f := cfg.fg(); f == nil {
		t.Fatal("empty TextConfig.fg() returned nil")
	}
}

func TestTextConfigExplicitLineSpacing(t *testing.T) {
	t.Parallel()

	cfg := &TextConfig{LineSpacing: 10}
	if got := cfg.lineSpacing(); got != 10 {
		t.Fatalf("TextConfig.lineSpacing() = %d, want 10", got)
	}
}

func TestTextConfigExplicitColors(t *testing.T) {
	t.Parallel()

	cfg := &TextConfig{
		Background: color.Black,
		Foreground: color.White,
	}
	if cfg.bg() == nil {
		t.Fatal("TextConfig.bg() returned nil for explicit color")
	}
	if cfg.fg() == nil {
		t.Fatal("TextConfig.fg() returned nil for explicit color")
	}
}

func TestTextConfigExplicitFace(t *testing.T) {
	t.Parallel()

	cfg := &TextConfig{Face: basicfont.Face7x13}
	if got := cfg.face(); got != basicfont.Face7x13 {
		t.Fatalf("TextConfig.face() returned unexpected face")
	}
}
