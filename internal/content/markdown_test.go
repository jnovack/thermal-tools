package content

import (
	"context"
	"strings"
	"testing"
)

// ── markdownToText unit tests ─────────────────────────────────────────────────

func TestMarkdownToTextParagraph(t *testing.T) {
	t.Parallel()

	text, err := markdownToText([]byte("Hello world."))
	if err != nil {
		t.Fatalf("markdownToText() error = %v", err)
	}
	if !strings.Contains(text, "Hello world") {
		t.Fatalf("markdownToText() = %q, want to contain 'Hello world'", text)
	}
}

func TestMarkdownToTextEmpty(t *testing.T) {
	t.Parallel()

	text, err := markdownToText([]byte(""))
	if err != nil {
		t.Fatalf("markdownToText(empty) error = %v", err)
	}
	if strings.TrimSpace(text) != "" {
		t.Fatalf("markdownToText(empty) = %q, want empty", text)
	}
}

func TestMarkdownToTextHeading(t *testing.T) {
	t.Parallel()

	text, err := markdownToText([]byte("# Title\n\nBody text."))
	if err != nil {
		t.Fatalf("markdownToText() error = %v", err)
	}
	if !strings.Contains(text, "Title") {
		t.Fatalf("markdownToText() = %q, want to contain heading text 'Title'", text)
	}
	// Heading should be followed by a separator line.
	if !strings.Contains(text, "---") {
		t.Fatalf("markdownToText() = %q, want separator after heading", text)
	}
}

func TestMarkdownToTextMultipleHeadingLevels(t *testing.T) {
	t.Parallel()

	md := "# H1\n\n## H2\n\n### H3"
	text, err := markdownToText([]byte(md))
	if err != nil {
		t.Fatalf("markdownToText() error = %v", err)
	}
	for _, want := range []string{"H1", "H2", "H3"} {
		if !strings.Contains(text, want) {
			t.Fatalf("markdownToText() = %q, want to contain %q", text, want)
		}
	}
}

func TestMarkdownToTextBulletList(t *testing.T) {
	t.Parallel()

	md := "- item one\n- item two\n- item three"
	text, err := markdownToText([]byte(md))
	if err != nil {
		t.Fatalf("markdownToText() error = %v", err)
	}
	for _, want := range []string{"item one", "item two", "item three"} {
		if !strings.Contains(text, want) {
			t.Fatalf("markdownToText() = %q, want to contain %q", text, want)
		}
	}
	// Items should be prefixed with bullet symbol.
	if !strings.Contains(text, "•") {
		t.Fatalf("markdownToText() = %q, want bullet symbol '•'", text)
	}
}

func TestMarkdownToTextFencedCodeBlock(t *testing.T) {
	t.Parallel()

	md := "```\ncode line one\ncode line two\n```"
	text, err := markdownToText([]byte(md))
	if err != nil {
		t.Fatalf("markdownToText() error = %v", err)
	}
	if !strings.Contains(text, "code line one") {
		t.Fatalf("markdownToText() = %q, want to contain code block content", text)
	}
}

func TestMarkdownToTextThematicBreak(t *testing.T) {
	t.Parallel()

	md := "above\n\n---\n\nbelow"
	text, err := markdownToText([]byte(md))
	if err != nil {
		t.Fatalf("markdownToText() error = %v", err)
	}
	if !strings.Contains(text, "above") || !strings.Contains(text, "below") {
		t.Fatalf("markdownToText() = %q, want text around break", text)
	}
	if !strings.Contains(text, "---") {
		t.Fatalf("markdownToText() = %q, want thematic break rendered as dashes", text)
	}
}

func TestMarkdownToTextInlineCode(t *testing.T) {
	t.Parallel()

	text, err := markdownToText([]byte("Use `fmt.Println` to print."))
	if err != nil {
		t.Fatalf("markdownToText() error = %v", err)
	}
	if !strings.Contains(text, "fmt.Println") {
		t.Fatalf("markdownToText() = %q, want to contain inline code content", text)
	}
}

// ── MarkdownToImage integration tests ─────────────────────────────────────────

func TestMarkdownToImageReturnsImage(t *testing.T) {
	t.Parallel()

	img, err := MarkdownToImage(context.Background(), []byte("Hello **world**."), 300)
	if err != nil {
		t.Fatalf("MarkdownToImage() error = %v", err)
	}
	if img == nil {
		t.Fatal("MarkdownToImage() returned nil image")
	}
}

func TestMarkdownToImageHasCorrectWidth(t *testing.T) {
	t.Parallel()

	img, err := MarkdownToImage(context.Background(), []byte("Some text that wraps"), 200)
	if err != nil {
		t.Fatalf("MarkdownToImage() error = %v", err)
	}
	if img.Bounds().Dx() != 200 {
		t.Fatalf("MarkdownToImage() width = %d, want 200", img.Bounds().Dx())
	}
}

func TestMarkdownToImageEmpty(t *testing.T) {
	t.Parallel()

	img, err := MarkdownToImage(context.Background(), []byte(""), 300)
	if err != nil {
		t.Fatalf("MarkdownToImage(empty) error = %v", err)
	}
	if img == nil {
		t.Fatal("MarkdownToImage(empty) returned nil")
	}
}

func TestMarkdownToImageNonZeroHeight(t *testing.T) {
	t.Parallel()

	md := []byte("Line one.\n\nLine two.\n\nLine three.")
	img, err := MarkdownToImage(context.Background(), md, 300)
	if err != nil {
		t.Fatalf("MarkdownToImage() error = %v", err)
	}
	if img.Bounds().Dy() == 0 {
		t.Fatal("MarkdownToImage() returned zero-height image")
	}
}

func TestMarkdownToImageContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel

	// MarkdownToImage uses the context only for the signature (not for
	// cancellation internally yet), so this should still succeed.
	img, err := MarkdownToImage(ctx, []byte("text"), 100)
	if err != nil {
		t.Fatalf("MarkdownToImage(cancelled ctx) error = %v", err)
	}
	if img == nil {
		t.Fatal("MarkdownToImage(cancelled ctx) returned nil")
	}
}
