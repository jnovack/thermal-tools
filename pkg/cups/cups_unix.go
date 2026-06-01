//go:build !windows

package cups

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// LPDriver prints via the CUPS lp(1) command on Unix/macOS.
type LPDriver struct {
	defaultPrinterName string
	media              string
	runner             CommandRunner
	// tempDir overrides os.TempDir() in tests.
	tempDir string
}

// New returns the CUPS lp driver for the current OS.
// printerName may be empty to enable auto-detection.
// media may be empty to use DefaultMedia.
func New(printerName, media string) Printer {
	return NewLPDriver(printerName, media)
}

// NewLPDriver constructs an LPDriver directly.
func NewLPDriver(printerName, media string) *LPDriver {
	if strings.TrimSpace(media) == "" {
		media = DefaultMedia
	}
	return &LPDriver{
		defaultPrinterName: printerName,
		media:              media,
		runner:             ExecRunner{},
	}
}

// Print sends data to the default (or auto-detected) printer.
func (d *LPDriver) Print(ctx context.Context, jobName string, data []byte) error {
	return d.PrintToPrinter(ctx, "", jobName, data)
}

// PrintToPrinter sends data to printerName (or auto-detected if empty).
func (d *LPDriver) PrintToPrinter(ctx context.Context, printerName, jobName string, data []byte) error {
	if _, err := exec.LookPath("lp"); err != nil {
		return fmt.Errorf("find lp command: %w", err)
	}

	dir := d.tempDir
	if dir == "" {
		dir = os.TempDir()
	}

	f, err := os.CreateTemp(dir, "thermal-tools-*.bin")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tempPath := f.Name()
	defer os.Remove(tempPath)

	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	args := []string{"-t", sanitizeJobName(jobName, "thermal-tools"), "-n", "1"}

	target, err := d.resolveTargetPrinter(ctx, printerName)
	if err != nil {
		return err
	}
	if target != "" {
		args = append(args, "-d", target)
	}
	if m := strings.TrimSpace(d.media); m != "" {
		args = append(args, "-o", "media="+m, "-o", "fit-to-page")
	}
	args = append(args, filepath.Clean(tempPath))

	if err := d.runner.Run(ctx, "lp", args...); err != nil {
		return fmt.Errorf("print with lp: %w", err)
	}
	return nil
}

// ListPrinters returns all printer names from lpstat -e.
func (d *LPDriver) ListPrinters(ctx context.Context) ([]string, error) {
	if _, err := exec.LookPath("lpstat"); err != nil {
		return nil, fmt.Errorf("find lpstat: %w", err)
	}
	cmd := exec.CommandContext(ctx, "lpstat", "-e")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("lpstat -e: %w", err)
	}
	return parseLines(string(out)), nil
}

// PreferredPrinterName returns the best thermal printer candidate.
func (d *LPDriver) PreferredPrinterName(ctx context.Context) (string, error) {
	if name := strings.TrimSpace(d.defaultPrinterName); name != "" {
		return name, nil
	}

	printers, err := d.ListPrinters(ctx)
	if err != nil {
		return "", err
	}

	type candidate struct {
		name  string
		score int
	}
	var scored []candidate
	for _, name := range printers {
		meta, _ := d.printerOptions(ctx, name)
		s := scorePrinterByMetadata(name, meta)
		scored = append(scored, candidate{name: name, score: s})
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].name < scored[j].name
	})

	if len(scored) > 0 && scored[0].score > 0 {
		return scored[0].name, nil
	}
	if len(printers) == 1 {
		return printers[0], nil
	}
	return "", nil
}

func (d *LPDriver) resolveTargetPrinter(ctx context.Context, printerName string) (string, error) {
	if name := strings.TrimSpace(printerName); name != "" {
		return name, nil
	}
	name, err := d.PreferredPrinterName(ctx)
	if err != nil {
		return "", fmt.Errorf("resolve printer: %w", err)
	}
	return name, nil
}

func (d *LPDriver) printerOptions(ctx context.Context, name string) (string, error) {
	if _, err := exec.LookPath("lpoptions"); err != nil {
		return "", fmt.Errorf("find lpoptions: %w", err)
	}
	cmd := exec.CommandContext(ctx, "lpoptions", "-p", name, "-l")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("lpoptions -p %q -l: %w", name, err)
	}
	return string(out), nil
}

func parseLines(s string) []string {
	raw := strings.Split(strings.ReplaceAll(s, "\r", ""), "\n")
	out := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, line := range raw {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}
