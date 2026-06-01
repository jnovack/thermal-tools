# pkg/cups

Cross-platform CUPS/LPD driver for sending print jobs to thermal (or any)
printers. Uses `lp(1)` on macOS/Linux and `mspaint /pt` on Windows. Exposes a
single `Printer` interface so the rest of your code does not need to branch on
OS.

## Import

```go
import "github.com/jnovack/thermal-tools/pkg/cups"
```

## Quickstart

```go
ctx := context.Background()

d := cups.New("", "")   // auto-detect printer, default media
if err := d.Print(ctx, "my-job", pngBytes); err != nil {
    return err
}
```

## Constructor

### `New(printerName, media string) Printer`

Returns the correct driver for the current OS (`LPDriver` backed by `lp` on
Unix, `mspaint` on Windows). Both arguments may be empty:

- `printerName` empty → auto-detection at print time (see below).
- `media` empty → `DefaultMedia` (`"Custom.60x30mm"`).

```go
// Explicit printer and label size
d := cups.New("Phomemo_M220", "Custom.70x30mm")

// Auto-detect everything
d := cups.New("", "")
```

### `NewLPDriver(printerName, media string) *LPDriver`

Returns the concrete `*LPDriver` instead of the `Printer` interface. Use this
when you need to inject a custom `CommandRunner` in tests.

## `Printer` interface

```go
type Printer interface {
    Print(ctx context.Context, jobName string, data []byte) error
    PrintToPrinter(ctx context.Context, printerName, jobName string, data []byte) error
    ListPrinters(ctx context.Context) ([]string, error)
    PreferredPrinterName(ctx context.Context) (string, error)
}
```

### `Print`

Sends `data` to the configured (or auto-detected) printer. `jobName` appears in
the CUPS queue and may be empty (falls back to `"thermal-tools"`).

```go
var buf bytes.Buffer
png.Encode(&buf, img)
if err := d.Print(ctx, "label-job", buf.Bytes()); err != nil {
    return err
}
```

### `PrintToPrinter`

Like `Print`, but overrides the printer name for this call only. Useful when
you want one driver instance but occasionally need to route to a specific
destination.

```go
if err := d.PrintToPrinter(ctx, "backup-printer", "job", data); err != nil {
    return err
}
```

### `ListPrinters`

Returns all printer names visible to the system (`lpstat -e` on Unix,
`Get-Printer` on Windows).

```go
names, err := d.ListPrinters(ctx)
for _, name := range names {
    fmt.Println(name)
}
```

### `PreferredPrinterName`

Returns the best thermal-printer candidate from the system's printer list.
If a printer name was given to `New`, that name is returned immediately. Otherwise,
printers are scored by name keywords and CUPS PPD metadata:

| Signal | Points |
| --- | --- |
| Name contains: `thermal`, `receipt`, `label`, `phomemo`, `m220`, `m110`, `pedoolo`, `482bt` | +3 each |
| PPD `MediaMethod/Method: Direct` | +5 |
| PPD `zeMediaTracking` or `MediaTracking` | +5 |
| PPD `203dpi` | +3 |
| PPD `Custom.WIDTHxHEIGHT` page size | +2 |
| PPD `Darkness/Darkness` option | +2 |
| PPD has `Letter Legal` and `Duplex` (office laser) | −4 |

If no printer scores above zero and exactly one printer exists, that printer is
returned anyway. An empty string is returned when no candidate is found.

```go
name, err := d.PreferredPrinterName(ctx)
if name == "" {
    // no thermal printer found
}
```

## Constants

```go
cups.DefaultMedia  // "Custom.60x30mm"
```

## Platform notes

### macOS / Linux

Uses `lp(1)` and `lpstat(1)`. The `lp` binary must be in `PATH`. Data is written
to a temp file and passed as a file argument to avoid shell injection.

```bash
lp -t "job-name" -n 1 -d printer-name -o media=Custom.60x30mm -o fit-to-page /tmp/thermal-tools-*.bin
```

### Windows

Uses `mspaint.exe /pt <file> [printer]` via `PowerShell Get-Printer` for
discovery. Data must be a PNG file — pass PNG bytes directly to `Print`.
Media size is not forwarded to mspaint (set the media in the printer driver
properties instead).

## Testing

`CommandRunner` is an interface, so you can inject a fake in tests without
spawning real processes:

```go
type fakeRunner struct{ called bool }

func (f *fakeRunner) Run(_ context.Context, _ string, _ ...string) error {
    f.called = true
    return nil
}

runner := &fakeRunner{}
d := &cups.LPDriver{} // unexported fields — use NewLPDriver, then replace runner via test helper
```

For integration tests that need a real `lp` binary, place a stub shell script
on `PATH`:

```bash
#!/bin/sh
exit 0
```

## Full example: render and print

```go
package main

import (
    "bytes"
    "context"
    "image/png"
    "log"

    "github.com/jnovack/thermal-tools/pkg/cups"
    "github.com/jnovack/thermal-tools/pkg/render"
)

func main() {
    ctx := context.Background()

    img, err := render.LoadImage("label.png")
    if err != nil {
        log.Fatal(err)
    }
    img = render.FitBox(img, 560, 243)
    img = render.Dither(img)

    var buf bytes.Buffer
    if err := png.Encode(&buf, img); err != nil {
        log.Fatal(err)
    }

    d := cups.New("", cups.DefaultMedia)
    if err := d.Print(ctx, "label-job", buf.Bytes()); err != nil {
        log.Fatal(err)
    }
}
```
