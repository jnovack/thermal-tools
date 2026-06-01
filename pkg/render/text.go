package render

import (
	"image"
	"image/color"
	"image/draw"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// TextConfig controls text rendering.
type TextConfig struct {
	// Face is the font face to use. Defaults to basicfont.Face7x13.
	Face font.Face

	// LineSpacing is the gap in pixels between lines. Defaults to 2.
	LineSpacing int

	// Background is the background color. Defaults to white.
	Background color.Color

	// Foreground is the text color. Defaults to black.
	Foreground color.Color
}

func (c *TextConfig) face() font.Face {
	if c != nil && c.Face != nil {
		return c.Face
	}
	return basicfont.Face7x13
}

func (c *TextConfig) lineSpacing() int {
	if c != nil && c.LineSpacing > 0 {
		return c.LineSpacing
	}
	return 2
}

func (c *TextConfig) bg() color.Color {
	if c != nil && c.Background != nil {
		return c.Background
	}
	return color.White
}

func (c *TextConfig) fg() color.Color {
	if c != nil && c.Foreground != nil {
		return c.Foreground
	}
	return color.Black
}

// Text renders text into an image no wider than width pixels, word-wrapping
// long lines. cfg may be nil to use defaults.
func Text(text string, width int, cfg *TextConfig) image.Image {
	face := cfg.face()
	spacing := cfg.lineSpacing()
	bg := cfg.bg()
	fg := cfg.fg()

	metrics := face.Metrics()
	lineH := metrics.Ascent.Round() + metrics.Descent.Round()

	lines := wrapText(text, face, width)
	if len(lines) == 0 {
		lines = []string{""}
	}

	imgH := len(lines)*lineH + (len(lines)-1)*spacing
	if imgH <= 0 {
		imgH = lineH
	}

	dst := image.NewRGBA(image.Rect(0, 0, width, imgH))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(bg), image.Point{}, draw.Src)

	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(fg),
		Face: face,
	}

	for i, line := range lines {
		y := i*(lineH+spacing) + metrics.Ascent.Round()
		d.Dot = fixed.P(0, y)
		d.DrawString(line)
	}
	return dst
}

// wrapText breaks text into lines that fit within width pixels.
func wrapText(text string, face font.Face, width int) []string {
	var out []string
	for _, paragraph := range strings.Split(text, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := ""
		for _, word := range words {
			candidate := word
			if line != "" {
				candidate = line + " " + word
			}
			if measureText(face, candidate) <= width {
				line = candidate
			} else {
				if line != "" {
					out = append(out, line)
				}
				line = word
			}
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func measureText(face font.Face, s string) int {
	return font.MeasureString(face, s).Round()
}
