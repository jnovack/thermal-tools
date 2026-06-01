# pkg/phomemo

BLE driver for Phomemo M220 label printers (also compatible with M110/M120).
The M220 is a 70mm wide, 203 DPI direct-thermal label printer. This package
handles BLE scanning, connection, and the full ESC/POS raster protocol.

## Import

```go
import "github.com/jnovack/thermal-tools/pkg/phomemo"
```

## Quickstart

```go
ctx := context.Background()

p, err := phomemo.Open(ctx, phomemo.DefaultDeviceName)
if err != nil {
    return fmt.Errorf("connect: %w", err)
}
defer p.Close()

// img must already be scaled and dithered — see pkg/render
if err := p.PrintImage(img); err != nil {
    return fmt.Errorf("print: %w", err)
}
```

## API

### `Open(ctx context.Context, name string) (*Printer, error)`

Scans for a Phomemo device by BLE advertised local name and connects. `name`
is matched exactly, or by the `"Mr.in"` prefix when `name` is
`DefaultDeviceName` (`"M220"`) — some firmware advertises as `"Mr.inM220"`.

Cancel `ctx` to abort the scan.

```go
p, err := phomemo.Open(ctx, "M220")
// or: phomemo.Open(ctx, "Mr.inM220")
```

### `(*Printer).Close() error`

Disconnects from the printer. Always defer this after a successful `Open`.

```go
p, err := phomemo.Open(ctx, name)
if err != nil { ... }
defer p.Close()
```

### `(*Printer).PrintImage(img image.Image) error`

Converts `img` to 1-bit-per-pixel and transmits it. Blocks until the printer
signals print-complete (up to 45 seconds for very tall labels).

The image must be prepared before calling:

1. Scale to `p.Width` pixels wide using `render.FitWidth` or `render.FitBox`.
2. Apply `render.Dither` or ensure the image is already 1-bit.

```go
img = render.FitWidth(img, p.Width)
img = render.Dither(img)
if err := p.PrintImage(img); err != nil {
    return err
}
```

## Printer fields

After `Open`, the returned `*Printer` exposes two settable fields:

| Field | Default | Description |
| --- | --- | --- |
| `Width` | `560` | Raster print width in pixels (70mm × 8 dots/mm) |
| `MediaType` | `MediaLabelWithGaps` | Label gap/mark detection mode |

```go
p.MediaType = phomemo.MediaContinuous  // for continuous rolls
```

## Media types

| Constant | Value | Use |
| --- | --- | --- |
| `MediaLabelWithGaps` | `0x0A` | Standard die-cut labels with gaps (default) |
| `MediaContinuous` | `0x0B` | Continuous rolls, no gap detection |
| `MediaLabelWithMarks` | `0x26` | Black-mark label stock |

## Constants

| Constant | Value | Description |
| --- | --- | --- |
| `DefaultDeviceName` | `"M220"` | BLE advertised local name to scan for |
| `PrintWidthPx` | `560` | Full roll width: 70mm × 8 dots/mm |
| `DPI` | `203` | Print resolution |

## Verbose mode

Set `phomemo.Verbose = true` before calling `Open` to log every BLE write and
notification to stderr in hex. Useful for debugging protocol issues.

```go
phomemo.Verbose = true
p, err := phomemo.Open(ctx, "M220")
```

## Hardware notes

These constraints are enforced inside the package and do not require caller attention, but are documented here for reference:

- **Write Without Response only.** Write Request causes silent failures on this family.
- **182-byte chunk limit.** Larger writes silently corrupt the print job.
- **60 ms inter-chunk pacing.** Required to avoid buffer overruns.
- **macOS hides MAC addresses.** Scanning always matches by `LocalName`, never by address.
- **Completion timeout.** `PrintImage` waits up to 45 seconds for the print-complete notification. A timeout is not treated as a fatal error.

## Full example: print an image over BLE

```go
package main

import (
    "context"
    "log"
    "os/signal"
    "syscall"

    "github.com/jnovack/thermal-tools/pkg/phomemo"
    "github.com/jnovack/thermal-tools/pkg/render"
)

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    p, err := phomemo.Open(ctx, phomemo.DefaultDeviceName)
    if err != nil {
        log.Fatalf("connect: %v", err)
    }
    defer p.Close()

    img, err := render.LoadImage("label.png")
    if err != nil {
        log.Fatalf("load: %v", err)
    }

    img = render.FitWidth(img, p.Width)
    img = render.Dither(img)

    if err := p.PrintImage(img); err != nil {
        log.Fatalf("print: %v", err)
    }
}
```
