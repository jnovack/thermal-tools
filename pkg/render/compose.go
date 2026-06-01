package render

import (
	"image"
	"image/color"
	"image/draw"
)

// VStack concatenates parts vertically with gap blank rows between each.
func VStack(gap int, parts ...image.Image) image.Image {
	if len(parts) == 0 {
		return image.NewGray(image.Rect(0, 0, 0, 0))
	}

	w := 0
	for _, p := range parts {
		if p.Bounds().Dx() > w {
			w = p.Bounds().Dx()
		}
	}

	totalH := 0
	for i, p := range parts {
		totalH += p.Bounds().Dy()
		if i < len(parts)-1 {
			totalH += gap
		}
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, totalH))
	// Fill white.
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)

	y := 0
	for i, p := range parts {
		b := p.Bounds()
		draw.Draw(dst, image.Rect(0, y, b.Dx(), y+b.Dy()), p, b.Min, draw.Over)
		y += b.Dy()
		if i < len(parts)-1 {
			y += gap
		}
	}
	return dst
}

// Rule returns a solid black horizontal bar of width w and height h pixels.
func Rule(w, h int) image.Image {
	img := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetGray(x, y, color.Gray{Y: 0})
		}
	}
	return img
}

// Solid returns a fully black rectangle of width w and height h pixels.
func Solid(w, h int) image.Image {
	return Rule(w, h)
}

// Uniform returns a rectangle filled with the given gray level (0=black, 255=white).
func Uniform(w, h int, level uint8) image.Image {
	img := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetGray(x, y, color.Gray{Y: level})
		}
	}
	return img
}

// Pad adds padTop blank white rows above img and padBottom below.
func Pad(img image.Image, padTop, padBottom int) image.Image {
	b := img.Bounds()
	dst := image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()+padTop+padBottom))
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(dst, image.Rect(0, padTop, b.Dx(), padTop+b.Dy()), img, b.Min, draw.Over)
	return dst
}

// Paginate splits img into vertical slices of pageHeight rows each, returning
// one image per page. Each page except possibly the last is exactly pageHeight
// rows tall; short pages are padded with white to reach pageHeight.
//
// If pageHeight is 0 or larger than img's height, a single-element slice
// containing the original image is returned (receipt / continuous mode).
func Paginate(img image.Image, pageHeight int) []image.Image {
	b := img.Bounds()
	h := b.Dy()
	w := b.Dx()

	if pageHeight <= 0 || pageHeight >= h {
		return []image.Image{img}
	}

	var pages []image.Image
	for y := b.Min.Y; y < b.Max.Y; y += pageHeight {
		end := y + pageHeight
		if end > b.Max.Y {
			end = b.Max.Y
		}

		// Always produce a full pageHeight-tall canvas, even for the last
		// (possibly short) slice, so the label feed is consistent.
		dst := image.NewRGBA(image.Rect(0, 0, w, pageHeight))
		draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
		draw.Draw(dst,
			image.Rect(0, 0, w, end-y),
			img,
			image.Point{b.Min.X, y},
			draw.Src,
		)
		pages = append(pages, dst)
	}
	return pages
}
