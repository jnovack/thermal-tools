package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"os/signal"
	"syscall"

	"github.com/jnovack/flag"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/jnovack/thermal-tools/internal/buildinfo"
	"github.com/jnovack/thermal-tools/internal/content"
	"github.com/jnovack/thermal-tools/pkg/cups"
	"github.com/jnovack/thermal-tools/pkg/phomemo"
	"github.com/jnovack/thermal-tools/pkg/render"
)

// Build-time ldflags populated by the Makefile.
var (
	version      = "dev"                  //nolint:unused
	buildRFC3339 = "1970-01-01T00:00:00Z" //nolint:unused
	revision     = "local"                //nolint:unused
)

func main() {
	buildinfo.PopulateFromBuildInfo()

	fs := flag.NewFlagSetWithEnvPrefix(os.Args[0], "", flag.ExitOnError)

	logLevel := fs.String("log-level", "info", "log level (debug|info|warn|error)")
	showVer := fs.Bool("version", false, "print version and exit")
	printer := fs.String("printer", "", "CUPS printer name (empty = auto-detect)")
	media := fs.String("media", cups.DefaultMedia, "CUPS media option (e.g. Custom.60x30mm)")
	bleName := fs.String("ble-name", "", "BLE device name for Phomemo M220; overrides CUPS")
	dryRun := fs.Bool("dry-run", false, "render to stdout as PNG without printing")
	widthMM := fs.Float64("width", 70.0, "label width in mm (default: 70 = M220 full roll width)")
	heightMM := fs.Float64("height", 0.0, "label height in mm (0 = continuous/receipt mode)")
	dpi := fs.Int("dpi", 203, "printer DPI used to convert mm to pixels")
	paginate := fs.Bool("paginate", false, "paginate content across multiple labels instead of scaling to fit")

	_ = fs.Parse(os.Args[1:])

	lvl, err := zerolog.ParseLevel(*logLevel)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr}).Level(lvl)

	if *showVer {
		fmt.Printf("thermal-tools %s (%s) built %s\n",
			buildinfo.Version, buildinfo.Revision, buildinfo.BuildRFC3339)
		os.Exit(0)
	}

	args := fs.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := printConfig{
		printerName: *printer,
		media:       *media,
		bleName:     *bleName,
		dryRun:      *dryRun,
		width:       mmToPx(*widthMM, *dpi),
		height:      mmToPx(*heightMM, *dpi),
		paginate:    *paginate,
	}

	var runErr error
	switch args[0] {
	case "image":
		runErr = cmdImage(ctx, args[1:], cfg)
	case "text":
		runErr = cmdText(ctx, args[1:], cfg)
	case "wifi":
		runErr = cmdWifi(ctx, args[1:], cfg)
	case "md", "markdown":
		runErr = cmdMarkdown(ctx, args[1:], cfg)
	case "printers":
		runErr = cmdPrinters(ctx)
	case "version":
		fmt.Printf("thermal-tools %s (%s) built %s\n",
			buildinfo.Version, buildinfo.Revision, buildinfo.BuildRFC3339)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		printUsage()
		os.Exit(1)
	}

	if runErr != nil {
		log.Fatal().Err(runErr).Msg("command failed")
	}
}

type printConfig struct {
	printerName string
	media       string
	bleName     string
	dryRun      bool
	// width and height are in pixels, derived from mm×dpi at flag-parse time.
	// height == 0 means continuous/receipt mode.
	width  int
	height int
	// paginate switches from the default scale-to-fit behaviour to splitting
	// content across multiple label-height pages.
	paginate bool
}

// mmToPx converts millimetres to pixels at the given DPI.
// Returns 0 for non-positive mm values.
func mmToPx(mm float64, dpi int) int {
	if mm <= 0 {
		return 0
	}
	return int(math.Round(mm * float64(dpi) / 25.4))
}

// printUsage writes the usage string to stderr.
func printUsage() {
	fmt.Fprintln(os.Stderr, `usage: thermal-tools [flags] <command> [args]

Commands:
  image  <file>              print an image file
  text   <text>              print text
  wifi   <ssid> <password>   print WiFi credentials as a QR code
  md     <file>              print a Markdown file
  printers                   list available CUPS printers
  version                    print version information

Global flags:
  --width      <mm>          label width in mm (default: 70 = M220 full roll width)
  --height     <mm>          label height in mm; 0 = continuous/receipt mode (default: 0)
  --dpi        <n>           printer DPI for mm→pixel conversion (default: 203)
  --paginate                 split content across multiple labels instead of scaling to fit
  --printer    <name>        CUPS printer name (default: auto-detect)
  --media      <size>        CUPS media option (default: Custom.60x30mm)
  --ble-name   <name>        BLE device name for Phomemo M220 (overrides CUPS)
  --dry-run                  write PNG to stdout instead of printing
  --log-level  <level>       debug|info|warn|error (default: info)
  --version                  print version and exit`)
}

// cmdImage prints an image file.
func cmdImage(ctx context.Context, args []string, cfg printConfig) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: thermal-tools image <file>")
	}
	img, err := render.LoadImage(args[0])
	if err != nil {
		return err
	}
	img = render.Dither(img)
	return emit(ctx, img, args[0], cfg)
}

// cmdText prints a string as text.
func cmdText(ctx context.Context, args []string, cfg printConfig) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: thermal-tools text <text>")
	}
	img := render.Text(args[0], cfg.width, nil)
	img = render.Dither(img)
	return emit(ctx, img, "text", cfg)
}

// cmdWifi prints WiFi credentials as a QR code.
func cmdWifi(ctx context.Context, args []string, cfg printConfig) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: thermal-tools wifi <ssid> <password>")
	}
	img, err := content.WifiQR(args[0], args[1], content.WifiWPA, false, cfg.width)
	if err != nil {
		return err
	}
	img = render.Dither(img)
	return emit(ctx, img, "wifi-qr", cfg)
}

// cmdMarkdown prints a Markdown file.
func cmdMarkdown(ctx context.Context, args []string, cfg printConfig) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: thermal-tools md <file>")
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return fmt.Errorf("read %q: %w", args[0], err)
	}
	img, err := content.MarkdownToImage(ctx, data, cfg.width)
	if err != nil {
		return err
	}
	img = render.Dither(img)
	return emit(ctx, img, args[0], cfg)
}

// cmdPrinters lists CUPS printers.
func cmdPrinters(ctx context.Context) error {
	d := cups.New("", "")
	printers, err := d.ListPrinters(ctx)
	if err != nil {
		return fmt.Errorf("list printers: %w", err)
	}
	if len(printers) == 0 {
		fmt.Println("no printers found")
		return nil
	}
	for _, p := range printers {
		fmt.Println(p)
	}
	return nil
}

// emit sizes img, optionally paginates it, then routes each page to the
// appropriate backend.
//
// Default (--paginate not set): FitBox scales content to fit within
// cfg.width×cfg.height in a single label (or FitWidth for receipt mode).
//
// With --paginate: content is fitted only to cfg.width, then split into
// cfg.height-tall pages — each page is a separate print job.
//
// Dry-run: all pages are VStacked into one PNG written to stdout.
func emit(ctx context.Context, img image.Image, jobName string, cfg printConfig) error {
	var pages []image.Image
	if cfg.paginate && cfg.height > 0 {
		// Paginate mode: fit width only, then split into label-height slices.
		img = render.FitWidth(img, cfg.width)
		pages = render.Paginate(img, cfg.height)
	} else {
		// Default: scale to fit the full label box (or just width when height=0).
		pages = []image.Image{render.FitBox(img, cfg.width, cfg.height)}
	}

	if cfg.dryRun {
		return png.Encode(os.Stdout, render.VStack(4, pages...))
	}

	if cfg.bleName != "" {
		p, err := phomemo.Open(ctx, cfg.bleName)
		if err != nil {
			return fmt.Errorf("connect to %q: %w", cfg.bleName, err)
		}
		defer p.Close()
		for i, page := range pages {
			log.Info().Int("page", i+1).Int("total", len(pages)).Msg("printing page")
			if err := p.PrintImage(page); err != nil {
				return fmt.Errorf("print page %d: %w", i+1, err)
			}
		}
		return nil
	}

	d := cups.New(cfg.printerName, cfg.media)
	for i, page := range pages {
		var buf bytes.Buffer
		if err := png.Encode(&buf, page); err != nil {
			return fmt.Errorf("encode page %d: %w", i+1, err)
		}
		log.Info().Int("page", i+1).Int("total", len(pages)).Msg("printing page")
		if err := d.Print(ctx, jobName, buf.Bytes()); err != nil {
			return fmt.Errorf("print page %d: %w", i+1, err)
		}
	}
	return nil
}
