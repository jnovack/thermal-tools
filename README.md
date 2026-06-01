# thermal-tools

A CLI and Go library for printing to thermal label and receipt printers.
Supports Phomemo M220 (and M110/M120) over BLE, and any CUPS/LPD printer on
macOS, Linux, and Windows.

## Requirements

- Go 1.25+
- **BLE printing**: macOS or Linux with Bluetooth LE support
- **CUPS printing**: macOS or Linux with CUPS installed (`lp`, `lpstat`); or Windows with PowerShell

## Install

```bash
go install github.com/jnovack/thermal-tools/cmd/thermal-tools@latest
```

Or build from source:

```bash
git clone https://github.com/jnovack/thermal-tools
cd thermal-tools
make build          # produces bin/thermal-tools
```

## Quickstart

### Print an image

```bash
# Auto-detect a CUPS thermal printer and print
thermal-tools image photo.png

# Scale to a specific label size
thermal-tools --width 70 --height 30 image photo.png

# Dry-run: write rendered PNG to stdout instead of printing
thermal-tools --dry-run image photo.png > preview.png
```

### Print text

```bash
thermal-tools text "Hello, world!"

# Word-wraps automatically to label width
thermal-tools --width 50 text "This is a longer message that will wrap."
```

### Print a WiFi QR code

```bash
thermal-tools wifi "MyNetwork" "MyPassword"
```

### Print a Markdown file

```bash
thermal-tools md notes.md
```

### Print via BLE (Phomemo M220)

```bash
# Scan for and connect to the M220 by its BLE advertised name
thermal-tools --ble-name M220 text "Hello from BLE"

# Some firmware advertises as "Mr.inM220"
thermal-tools --ble-name Mr.inM220 image logo.png
```

### List available CUPS printers

```bash
thermal-tools printers
```

## Flags

| Flag | Default | Description |
| --- | --- | --- |
| `--width` | `70.0` | Label width in mm |
| `--height` | `0.0` | Label height in mm; `0` = continuous/receipt mode |
| `--dpi` | `203` | Printer DPI for mm→pixel conversion |
| `--paginate` | off | Split tall content across multiple labels instead of scaling to fit |
| `--printer` | auto | CUPS printer name; empty enables auto-detection |
| `--media` | `Custom.60x30mm` | CUPS media option passed to `lp -o media=` |
| `--ble-name` | — | BLE advertised name of Phomemo device; overrides CUPS |
| `--dry-run` | off | Render to PNG on stdout, do not send to printer |
| `--log-level` | `info` | Log verbosity: `debug`, `info`, `warn`, `error` |
| `--version` | — | Print version and exit |

## Commands

| Command | Usage | Description |
| --- | --- | --- |
| `image` | `image <file>` | Print an image file (PNG, JPEG, BMP, WebP, TIFF, GIF) |
| `text` | `text <text>` | Print plain text, word-wrapped to label width |
| `wifi` | `wifi <ssid> <password>` | Print WiFi credentials as a QR code |
| `md` | `md <file>` | Render and print a Markdown file |
| `printers` | `printers` | List CUPS printers visible to the system |
| `version` | `version` | Print version, revision, and build timestamp |

## Paginate mode

By default, content is scaled to fit the label dimensions. With `--paginate`,
content is fitted to `--width` only and then split into `--height`-tall pages,
each sent as a separate print job:

```bash
thermal-tools --width 70 --height 30 --paginate md long-doc.md
```

## Go library

thermal-tools is also a set of importable packages:

| Package | Description |
| --- | --- |
| [`pkg/render`](pkg/render/README.md) | Image pipeline: load, scale, rotate, dither, compose, encode |
| [`pkg/phomemo`](pkg/phomemo/README.md) | BLE driver for Phomemo M220/M110/M120 |
| [`pkg/cups`](pkg/cups/README.md) | CUPS/LPD driver for macOS, Linux, and Windows |

See each package README for integration details and code examples.
