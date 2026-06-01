//go:build windows

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

// LPDriver prints via mspaint /pt on Windows and enumerates printers via PowerShell.
type LPDriver struct {
	defaultPrinterName string
	media              string
	runner             CommandRunner
	tempDir            string
}

// New returns the Windows mspaint driver.
func New(printerName, media string) Printer {
	return NewLPDriver(printerName, media)
}

// NewLPDriver constructs an LPDriver for Windows.
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
// On Windows the data must be a PNG file — pass image bytes directly.
func (d *LPDriver) PrintToPrinter(ctx context.Context, printerName, jobName string, data []byte) error {
	dir := d.tempDir
	if dir == "" {
		dir = os.TempDir()
	}

	f, err := os.CreateTemp(dir, "thermal-tools-*.png")
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

	target, err := d.resolveTargetPrinter(ctx, printerName)
	if err != nil {
		return err
	}

	args := []string{"/pt", filepath.Clean(tempPath)}
	if target != "" {
		args = append(args, target)
	}

	if err := d.runner.Run(ctx, "mspaint.exe", args...); err != nil {
		return fmt.Errorf("print with mspaint: %w", err)
	}
	return nil
}

// ListPrinters returns printer names via PowerShell Get-Printer.
func (d *LPDriver) ListPrinters(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command",
		"Get-Printer | Select-Object -ExpandProperty Name")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Get-Printer: %w", err)
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
		s := scorePrinterByMetadata(name, "")
		if s > 0 {
			scored = append(scored, candidate{name: name, score: s})
		}
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].name < scored[j].name
	})

	if len(scored) > 0 {
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
