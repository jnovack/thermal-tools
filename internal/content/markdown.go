package content

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"

	"github.com/jnovack/thermal-tools/pkg/render"
)

// MarkdownToImage parses md as Markdown and renders it to an image of
// width pixels. Headings, paragraphs, code blocks, and lists are rendered
// as plain text with visual separators.
func MarkdownToImage(_ context.Context, md []byte, width int) (image.Image, error) {
	plain, err := markdownToText(md)
	if err != nil {
		return nil, fmt.Errorf("render markdown: %w", err)
	}
	return render.Text(plain, width, nil), nil
}

// markdownToText converts Markdown bytes to plain text suitable for
// line-based rendering. Headings get a trailing separator line.
func markdownToText(src []byte) (string, error) {
	parser := goldmark.DefaultParser()
	reader := text.NewReader(src)
	doc := parser.Parse(reader)

	var buf bytes.Buffer
	if err := walkNode(doc, src, &buf, 0); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

func walkNode(n ast.Node, src []byte, buf *bytes.Buffer, depth int) error {
	switch node := n.(type) {
	case *ast.Heading:
		prefix := strings.Repeat("#", node.Level) + " "
		buf.WriteString(prefix)
		if err := walkChildren(n, src, buf, depth+1); err != nil {
			return err
		}
		buf.WriteString("\n")
		buf.WriteString(strings.Repeat("-", 24) + "\n")
	case *ast.Paragraph:
		if err := walkChildren(n, src, buf, depth+1); err != nil {
			return err
		}
		buf.WriteString("\n\n")
	case *ast.Text:
		buf.Write(node.Segment.Value(src))
		if node.HardLineBreak() {
			buf.WriteString("\n")
		} else if node.SoftLineBreak() {
			buf.WriteString(" ")
		}
	case *ast.CodeSpan:
		buf.WriteString("`")
		if err := walkChildren(n, src, buf, depth+1); err != nil {
			return err
		}
		buf.WriteString("`")
	case *ast.FencedCodeBlock, *ast.CodeBlock:
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			buf.Write(line.Value(src))
		}
		buf.WriteString("\n")
	case *ast.List:
		if err := walkChildren(n, src, buf, depth+1); err != nil {
			return err
		}
		buf.WriteString("\n")
	case *ast.ListItem:
		if depth > 0 {
			buf.WriteString(strings.Repeat("  ", depth-1))
		}
		buf.WriteString("• ")
		if err := walkChildren(n, src, buf, depth+1); err != nil {
			return err
		}
	case *ast.ThematicBreak:
		buf.WriteString(strings.Repeat("-", 24) + "\n")
	case *ast.Blockquote:
		if err := walkChildren(n, src, buf, depth+1); err != nil {
			return err
		}
	default:
		if err := walkChildren(n, src, buf, depth); err != nil {
			return err
		}
	}
	return nil
}

func walkChildren(n ast.Node, src []byte, buf *bytes.Buffer, depth int) error {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if err := walkNode(child, src, buf, depth); err != nil {
			return err
		}
	}
	return nil
}
