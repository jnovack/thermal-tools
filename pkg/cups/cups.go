// Package cups provides a cross-platform interface for sending data to
// printers via the CUPS/LPD subsystem on Unix and mspaint on Windows.
//
// The exported Printer interface is platform-agnostic. Call New to get
// the correct driver for the current OS.
package cups

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// DefaultMedia is the default CUPS media option for thermal label printers.
const DefaultMedia = "Custom.60x30mm"

// Printer is the platform-agnostic interface for sending data to a printer.
type Printer interface {
	// Print sends data to the default (or auto-detected) printer.
	Print(ctx context.Context, jobName string, data []byte) error

	// PrintToPrinter sends data to the named printer, or the auto-detected
	// printer if printerName is empty.
	PrintToPrinter(ctx context.Context, printerName, jobName string, data []byte) error

	// ListPrinters returns the names of all available printers.
	ListPrinters(ctx context.Context) ([]string, error)

	// PreferredPrinterName returns the best-scoring thermal printer name,
	// or the single available printer if only one exists. Returns empty
	// string when no good candidate is found.
	PreferredPrinterName(ctx context.Context) (string, error)
}

// CommandRunner abstracts exec.Command so drivers can be tested without
// running real system commands.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) error
}

// ExecRunner is the production CommandRunner using os/exec.
type ExecRunner struct{}

// Run executes name with args, returning combined output in the error on failure.
func (ExecRunner) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(output)); msg != "" {
			return fmt.Errorf("%s: %w", msg, err)
		}
		return err
	}
	return nil
}

// scorePrinterByMetadata scores a printer by its name and lpoptions metadata.
// Positive scores indicate a thermal label printer; negative scores indicate
// an unlikely candidate (e.g. a networked laser printer).
func scorePrinterByMetadata(name, metadata string) int {
	score := 0
	lowerName := strings.ToLower(name)
	lowerMeta := strings.ToLower(metadata)

	for _, needle := range []string{"thermal", "receipt", "label", "pedoolo", "by-482bt", "482bt", "phomemo", "m220", "m110"} {
		if strings.Contains(lowerName, needle) {
			score += 3
		}
	}

	// CUPS PPD attributes that identify direct-thermal label printers.
	if strings.Contains(lowerMeta, "mediamethod/method") && strings.Contains(lowerMeta, "direct") {
		score += 5
	}
	if strings.Contains(lowerMeta, "mediatracking") || strings.Contains(lowerMeta, "zemediatracking") {
		score += 5
	}
	if strings.Contains(lowerMeta, "203dpi") {
		score += 3
	}
	if strings.Contains(lowerMeta, "custom.widthxheight") {
		score += 2
	}
	if strings.Contains(lowerMeta, "darkness/darkness") {
		score += 2
	}

	// Penalise multifunction office printers.
	if strings.Contains(lowerMeta, "letter legal") && strings.Contains(lowerMeta, "duplex") {
		score -= 4
	}

	return score
}

func sanitizeJobName(name, fallback string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return fallback
	}
	return strings.ReplaceAll(name, "\n", " ")
}
