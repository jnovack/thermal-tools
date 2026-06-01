// Package render provides image processing utilities for thermal printers.
// All functions are pure — no I/O, no side effects beyond the returned image.
package render

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"os"

	xdraw "golang.org/x/image/draw"

	// Register extended format decoders for image.Decode.
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

// LoadImage opens and decodes an image file.
// Supports PNG, JPEG, GIF, BMP, WebP, and TIFF.
func LoadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}
	defer f.Close()
	return DecodeImage(f)
}

// DecodeImage decodes an image from r.
// The format is detected automatically.
func DecodeImage(r io.Reader) (image.Image, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}
	return img, nil
}

// FitWidth resizes img so its width equals targetWidth pixels, preserving
// the aspect ratio. Uses Catmull-Rom for smooth downscaling.
func FitWidth(img image.Image, targetWidth int) image.Image {
	b := img.Bounds()
	srcW := b.Dx()
	srcH := b.Dy()
	if srcW == 0 || targetWidth == 0 {
		return img
	}
	newH := srcH * targetWidth / srcW
	if newH == 0 {
		newH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, newH))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, b, xdraw.Over, nil)
	return dst
}

// FitBox scales img to fit within width×height pixels while maintaining
// aspect ratio, centering the result on a white canvas of exactly
// width×height. The image may be scaled up or down as needed to fill
// the label.
//
// If height is 0, only width is constrained — equivalent to FitWidth.
func FitBox(img image.Image, width, height int) image.Image {
	if height <= 0 {
		return FitWidth(img, width)
	}
	b := img.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	if srcW == 0 || srcH == 0 || width == 0 || height == 0 {
		return image.NewRGBA(image.Rect(0, 0, width, height))
	}

	scale := math.Min(float64(width)/float64(srcW), float64(height)/float64(srcH))
	newW := int(math.Round(float64(srcW) * scale))
	newH := int(math.Round(float64(srcH) * scale))
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	scaled := image.NewRGBA(image.Rect(0, 0, newW, newH))
	xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), img, b, xdraw.Over, nil)

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	ox := (width - newW) / 2
	oy := (height - newH) / 2
	draw.Draw(dst, image.Rect(ox, oy, ox+newW, oy+newH), scaled, image.Point{}, draw.Over)
	return dst
}

// Dither converts img to a 1-bit black-and-white paletted image using
// Floyd-Steinberg error diffusion. The returned image has exactly two
// palette entries: white (index 0) and black (index 1).
func Dither(img image.Image) *image.Paletted {
	b := img.Bounds()
	palette := color.Palette{color.White, color.Black}
	dst := image.NewPaletted(b, palette)
	draw.FloydSteinberg.Draw(dst, b, img, b.Min)
	return dst
}

// ToGray converts img to an 8-bit grayscale image without dithering.
// Useful as a pre-processing step before Dither.
func ToGray(img image.Image) *image.Gray {
	b := img.Bounds()
	dst := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(x, y, color.GrayModel.Convert(img.At(x, y)))
		}
	}
	return dst
}

// Rotate90 rotates img 90 degrees clockwise.
func Rotate90(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(b.Max.Y-1-y, x, img.At(x, y))
		}
	}
	return dst
}

// Rotate180 rotates img 180 degrees.
func Rotate180(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(b)
	maxX := b.Max.X - 1
	maxY := b.Max.Y - 1
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(maxX-x, maxY-y, img.At(x, y))
		}
	}
	return dst
}

// Rotate270 rotates img 270 degrees clockwise (90 counter-clockwise).
func Rotate270(img image.Image) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dy(), b.Dx()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(y, b.Max.X-1-x, img.At(x, y))
		}
	}
	return dst
}

// EncodePNG encodes img as PNG bytes.
func EncodePNG(img image.Image) ([]byte, error) {
	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		errCh <- png.Encode(pw, img)
		pw.Close()
	}()
	data, err := io.ReadAll(pr)
	if err != nil {
		return nil, fmt.Errorf("encode PNG: %w", err)
	}
	if err := <-errCh; err != nil {
		return nil, fmt.Errorf("encode PNG: %w", err)
	}
	return data, nil
}

// EncodeJPEG encodes img as JPEG bytes at quality q (1–100).
func EncodeJPEG(img image.Image, q int) ([]byte, error) {
	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		errCh <- jpeg.Encode(pw, img, &jpeg.Options{Quality: q})
		pw.Close()
	}()
	data, err := io.ReadAll(pr)
	if err != nil {
		return nil, fmt.Errorf("encode JPEG: %w", err)
	}
	if err := <-errCh; err != nil {
		return nil, fmt.Errorf("encode JPEG: %w", err)
	}
	return data, nil
}
