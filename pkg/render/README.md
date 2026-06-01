# pkg/render

Image processing pipeline for thermal printers. All functions are pure — no
I/O, no global state, no side effects beyond the returned value. Designed to
be composed: load → scale → rotate → compose → dither → encode.

## Import

```go
import "github.com/jnovack/thermal-tools/pkg/render"
```

## Typical pipeline

```go
// Load an image from disk, scale it to the printer width, dither to 1-bit,
// and encode as PNG for transmission.
img, err := render.LoadImage("photo.jpg")
if err != nil {
    return err
}
img = render.FitWidth(img, 560)      // M220 is 560 px wide
img = render.Dither(img)             // Floyd-Steinberg to 1-bit
data, err := render.EncodePNG(img)
```

## Loading images

### `LoadImage(path string) (image.Image, error)`

Opens and decodes an image file. Supports PNG, JPEG, GIF, BMP, WebP, and TIFF.

```go
img, err := render.LoadImage("label.png")
```

### `DecodeImage(r io.Reader) (image.Image, error)`

Decodes an image from any `io.Reader`. Format is auto-detected.

```go
img, err := render.DecodeImage(resp.Body)
```

## Scaling

### `FitWidth(img image.Image, targetWidth int) image.Image`

Resizes `img` so its width equals `targetWidth`, preserving the aspect ratio.
Uses Catmull-Rom for smooth downscaling.

```go
img = render.FitWidth(img, 560)     // scale to M220 width
```

### `FitBox(img image.Image, width, height int) image.Image`

Scales `img` to fit within `width×height` while maintaining aspect ratio,
centering it on a white canvas of exactly `width×height`. If `height` is `0`,
behaves identically to `FitWidth`.

```go
// Scale to fit a 70×30mm label at 203 DPI (560×243 px)
img = render.FitBox(img, 560, 243)
```

## Rotation

All rotate functions return a new image; the source is unchanged.

| Function | Effect |
| --- | --- |
| `Rotate90(img)` | 90° clockwise |
| `Rotate180(img)` | 180° |
| `Rotate270(img)` | 270° clockwise (90° counter-clockwise) |

```go
img = render.Rotate90(img)
```

## Composing

### `VStack(gap int, parts ...image.Image) image.Image`

Stacks images vertically with `gap` blank white rows between each part. The
canvas width equals the widest input; narrower images are left-aligned.

```go
header := render.Text("ORDER #1234", 560, nil)
divider := render.Rule(560, 2)
body := render.Text(itemList, 560, nil)
combined := render.VStack(8, header, divider, body)
```

### `Rule(w, h int) image.Image`

Returns a solid black horizontal bar `w` pixels wide and `h` pixels tall.
Useful as a visual separator.

### `Solid(w, h int) image.Image`

Alias for `Rule`. Returns a fully black rectangle.

### `Uniform(w, h int, level uint8) image.Image`

Returns a rectangle filled with a uniform gray level: `0` = black, `255` = white.

```go
bg := render.Uniform(560, 20, 200) // light gray band
```

### `Pad(img image.Image, padTop, padBottom int) image.Image`

Adds `padTop` blank white rows above and `padBottom` below.

```go
img = render.Pad(img, 10, 10)
```

### `Paginate(img image.Image, pageHeight int) []image.Image`

Splits `img` into vertical slices of `pageHeight` rows, one per label. The
last page is padded with white to reach `pageHeight`. If `pageHeight` is `0`
or larger than the image height, returns a single-element slice (receipt mode).

```go
pages := render.Paginate(img, 243)  // split into 30mm pages at 203 DPI
for _, page := range pages {
    printer.PrintImage(page)
}
```

## Rendering text

### `Text(text string, width int, cfg *TextConfig) image.Image`

Renders `text` into an image `width` pixels wide, word-wrapping long lines.
`cfg` may be `nil` to use defaults (7×13 bitmap font, 2px line spacing,
white background, black foreground).

```go
img := render.Text("Hello, world!", 560, nil)
```

### `TextConfig`

```go
cfg := &render.TextConfig{
    Face:        myFontFace,     // golang.org/x/image/font.Face; nil = default
    LineSpacing: 4,              // pixels between lines; 0 = default (2)
    Background:  color.White,   // nil = white
    Foreground:  color.Black,   // nil = black
}
img := render.Text("Hello", 560, cfg)
```

## Dithering and grayscale

### `Dither(img image.Image) *image.Paletted`

Converts to 1-bit black-and-white using Floyd-Steinberg error diffusion.
The returned palette has two entries: white (index 0) and black (index 1).
Always call this as the last step before encoding or transmitting.

```go
dithered := render.Dither(img)
```

### `ToGray(img image.Image) *image.Gray`

Converts to 8-bit grayscale without dithering. Useful as a pre-processing step
before `Dither` when the source is a color image.

```go
gray := render.ToGray(colorImg)
dithered := render.Dither(gray)
```

## Encoding

### `EncodePNG(img image.Image) ([]byte, error)`

Encodes to PNG bytes. Lossless and recommended for transmission to printers.

```go
data, err := render.EncodePNG(dithered)
```

### `EncodeJPEG(img image.Image, q int) ([]byte, error)`

Encodes to JPEG bytes at quality `q` (1–100).

```go
data, err := render.EncodeJPEG(img, 85)
```

## Full example: label with header and body

```go
package main

import (
    "image/color"
    "log"
    "os"

    "github.com/jnovack/thermal-tools/pkg/render"
)

func main() {
    const width = 560 // M220: 70mm × 8 dots/mm

    header := render.Text("RECEIPT", width, &render.TextConfig{
        Background: color.Black,
        Foreground: color.White,
    })
    header = render.Pad(header, 4, 4)

    rule := render.Rule(width, 2)

    body := render.Text("Item 1 ........... $4.99\nItem 2 ........... $2.50\nTotal ............. $7.49", width, nil)

    label := render.VStack(6, header, rule, body)
    label = render.Dither(label)

    data, err := render.EncodePNG(label)
    if err != nil {
        log.Fatal(err)
    }
    os.Stdout.Write(data)
}
```
