//go:build !windows

package cups

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// recordingRunner captures the name and args of the most recent Run call
// without executing any real process.
type recordingRunner struct {
	name string
	args []string
	err  error // err, if non-nil, is returned by Run
}

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) error {
	r.name = name
	r.args = append([]string(nil), args...)
	return r.err
}

// stubLP writes a no-op lp shell script into dir and prepends dir to PATH.
// It returns the original PATH value and a cleanup function.
func stubLP(t *testing.T, dir string) {
	t.Helper()
	lpPath := filepath.Join(dir, "lp")
	if err := os.WriteFile(lpPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write stub lp: %v", err)
	}
	origPath := os.Getenv("PATH")
	if err := os.Setenv("PATH", dir+string(os.PathListSeparator)+origPath); err != nil {
		t.Fatalf("Setenv PATH: %v", err)
	}
	t.Cleanup(func() { _ = os.Setenv("PATH", origPath) })
}

// ── Happy-path print tests ───────────────────────────────────────────────────

func TestLPDriverPrint(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	stubLP(t, tempDir)

	runner := &recordingRunner{}
	driver := &LPDriver{
		defaultPrinterName: "pedoolo",
		media:              "Custom.80x60mm",
		runner:             runner,
		tempDir:            tempDir,
	}

	if err := driver.Print(context.Background(), "Test Job", []byte("data")); err != nil {
		t.Fatalf("Print() error = %v", err)
	}
	if runner.name != "lp" {
		t.Fatalf("runner name = %q, want lp", runner.name)
	}
	if len(runner.args) < 2 || runner.args[0] != "-t" || runner.args[1] != "Test Job" {
		t.Fatalf("unexpected job name args: %v", runner.args)
	}
}

func TestLPDriverPrintToPrinterExplicit(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	stubLP(t, tempDir)

	runner := &recordingRunner{}
	driver := &LPDriver{runner: runner, tempDir: tempDir}

	if err := driver.PrintToPrinter(context.Background(), "my-printer", "job", []byte("data")); err != nil {
		t.Fatalf("PrintToPrinter() error = %v", err)
	}

	found := false
	for i, a := range runner.args {
		if a == "-d" && i+1 < len(runner.args) && runner.args[i+1] == "my-printer" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("args %v do not contain '-d my-printer'", runner.args)
	}
}

func TestLPDriverPrintWithMedia(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	stubLP(t, tempDir)

	runner := &recordingRunner{}
	driver := &LPDriver{
		media:   "Custom.40x30mm",
		runner:  runner,
		tempDir: tempDir,
	}

	if err := driver.Print(context.Background(), "job", []byte("data")); err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	var hasMedia bool
	for _, a := range runner.args {
		if strings.Contains(a, "Custom.40x30mm") {
			hasMedia = true
			break
		}
	}
	if !hasMedia {
		t.Fatalf("args %v do not contain media option", runner.args)
	}
}

func TestLPDriverPrintNoMedia(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	stubLP(t, tempDir)

	runner := &recordingRunner{}
	driver := &LPDriver{runner: runner, media: "", tempDir: tempDir}

	if err := driver.Print(context.Background(), "job", []byte("data")); err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	for i, a := range runner.args {
		if a == "-o" && i+1 < len(runner.args) && strings.HasPrefix(runner.args[i+1], "media=") {
			t.Fatalf("expected no media arg but got %v", runner.args)
		}
	}
}

func TestLPDriverPrintSanitizesJobName(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	stubLP(t, tempDir)

	runner := &recordingRunner{}
	driver := &LPDriver{runner: runner, tempDir: tempDir}

	if err := driver.Print(context.Background(), "line1\nline2", []byte("data")); err != nil {
		t.Fatalf("Print() error = %v", err)
	}
	if len(runner.args) < 2 || runner.args[0] != "-t" {
		t.Fatalf("unexpected args: %v", runner.args)
	}
	if strings.Contains(runner.args[1], "\n") {
		t.Fatalf("job name %q still contains newline", runner.args[1])
	}
}

func TestLPDriverPrintEmptyJobName(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	stubLP(t, tempDir)

	runner := &recordingRunner{}
	driver := &LPDriver{runner: runner, tempDir: tempDir}

	if err := driver.Print(context.Background(), "", []byte("data")); err != nil {
		t.Fatalf("Print() error = %v", err)
	}
	if len(runner.args) < 2 || runner.args[0] != "-t" || runner.args[1] == "" {
		t.Fatalf("expected non-empty fallback job name, got args: %v", runner.args)
	}
}

// ── Error-path print tests ───────────────────────────────────────────────────

func TestLPDriverPrintLpNotFound(t *testing.T) {
	// Not parallel — t.Setenv cannot be used with t.Parallel.
	t.Setenv("PATH", t.TempDir())

	driver := NewLPDriver("printer", "")
	err := driver.Print(context.Background(), "job", []byte("data"))
	if err == nil {
		t.Fatal("Print() expected error when lp is not in PATH, got nil")
	}
	if !strings.Contains(err.Error(), "lp") {
		t.Fatalf("error %q should mention 'lp'", err.Error())
	}
}

func TestLPDriverPrintRunnerError(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	stubLP(t, tempDir)

	sentinel := errors.New("lp exploded")
	runner := &recordingRunner{err: sentinel}
	driver := &LPDriver{runner: runner, tempDir: tempDir}

	err := driver.Print(context.Background(), "job", []byte("data"))
	if err == nil {
		t.Fatal("Print() expected error from runner, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("error %v does not wrap sentinel %v", err, sentinel)
	}
}

// ── Printer discovery tests ──────────────────────────────────────────────────

func TestLPDriverListPrintersLpstatNotFound(t *testing.T) {
	// Not parallel — t.Setenv cannot be used with t.Parallel.
	t.Setenv("PATH", t.TempDir())

	driver := NewLPDriver("", "")
	_, err := driver.ListPrinters(context.Background())
	if err == nil {
		t.Fatal("ListPrinters() expected error when lpstat is not in PATH, got nil")
	}
}

// ── Constructor tests ────────────────────────────────────────────────────────

func TestNewLPDriverDefaultMedia(t *testing.T) {
	t.Parallel()

	d := NewLPDriver("", "")
	if d.media != DefaultMedia {
		t.Fatalf("empty media → got %q, want %q", d.media, DefaultMedia)
	}
}

func TestNewLPDriverCustomMedia(t *testing.T) {
	t.Parallel()

	d := NewLPDriver("printer", "Custom.40x30mm")
	if d.media != "Custom.40x30mm" {
		t.Fatalf("custom media → got %q, want Custom.40x30mm", d.media)
	}
}

func TestNewLPDriverWhitespaceMedia(t *testing.T) {
	t.Parallel()

	d := NewLPDriver("", "   ")
	if d.media != DefaultMedia {
		t.Fatalf("whitespace media → got %q, want %q", d.media, DefaultMedia)
	}
}

// ── parseLines tests ─────────────────────────────────────────────────────────

func TestParseLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "unix newlines",
			input: "printer-a\nprinter-b\nprinter-c",
			want:  []string{"printer-a", "printer-b", "printer-c"},
		},
		{
			name:  "windows CRLF",
			input: "printer-a\r\nprinter-b\r\n",
			want:  []string{"printer-a", "printer-b"},
		},
		{
			name:  "empty lines stripped",
			input: "printer-a\n\n\nprinter-b",
			want:  []string{"printer-a", "printer-b"},
		},
		{
			name:  "duplicates removed",
			input: "printer-a\nprinter-a\nprinter-b",
			want:  []string{"printer-a", "printer-b"},
		},
		{
			name:  "leading and trailing whitespace trimmed",
			input: "  printer-a  \n  printer-b  ",
			want:  []string{"printer-a", "printer-b"},
		},
		{
			name:  "empty input",
			input: "",
			want:  []string{},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := parseLines(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("parseLines() = %v (len %d), want %v (len %d)",
					got, len(got), tc.want, len(tc.want))
			}
			for i, g := range got {
				if g != tc.want[i] {
					t.Fatalf("parseLines()[%d] = %q, want %q", i, g, tc.want[i])
				}
			}
		})
	}
}

// ── scorePrinterByMetadata tests ─────────────────────────────────────────────

func TestScorePrinterByMetadata(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		printer  string
		metadata string
		wantPos  bool
	}{
		{
			name:    "thermal label printer scores positive",
			printer: "_BY_482BT",
			metadata: "PageSize/Media Size: Custom.WIDTHxHEIGHT\n" +
				"Resolution/Resolution: *203dpi\n" +
				"zeMediaTracking/Media Tracking: Continuous *Gap BLine\n" +
				"MediaMethod/Method: *Normal Direct\n" +
				"Darkness/Darkness: *8\n",
			wantPos: true,
		},
		{
			name:    "office laser printer scores non-positive",
			printer: "HP_LaserJet_1320",
			metadata: "PageSize/Media Size: *Letter Legal A4\n" +
				"Duplex/2-Sided Printing: *None DuplexNoTumble DuplexTumble\n",
			wantPos: false,
		},
		{
			name:     "pedoolo name alone scores positive",
			printer:  "Pedoolo_Printer",
			metadata: "",
			wantPos:  true,
		},
		{
			name:     "phomemo name scores positive",
			printer:  "Phomemo_M220",
			metadata: "",
			wantPos:  true,
		},
		{
			name:     "unknown generic printer",
			printer:  "Generic_Printer",
			metadata: "PageSize/Media Size: *A4\n",
			wantPos:  false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := scorePrinterByMetadata(tc.printer, tc.metadata)
			if tc.wantPos && got <= 0 {
				t.Fatalf("scorePrinterByMetadata(%q, ...) = %d, want > 0", tc.printer, got)
			}
			if !tc.wantPos && got > 0 {
				t.Fatalf("scorePrinterByMetadata(%q, ...) = %d, want <= 0", tc.printer, got)
			}
		})
	}
}

// ── sanitizeJobName tests ─────────────────────────────────────────────────────

func TestSanitizeJobName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		fallback string
		want     string
	}{
		{"empty uses fallback", "", "thermal-tools", "thermal-tools"},
		{"whitespace-only uses fallback", "   ", "thermal-tools", "thermal-tools"},
		{"trims surrounding space", "  hello  ", "fb", "hello"},
		{"replaces newline", "foo\nbar", "fb", "foo bar"},
		{"replaces CRLF newline", "foo\r\nbar", "fb", "foo\r bar"},
		{"non-empty passthrough", "My Print Job", "fb", "My Print Job"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := sanitizeJobName(tc.input, tc.fallback); got != tc.want {
				t.Fatalf("sanitizeJobName(%q, %q) = %q, want %q",
					tc.input, tc.fallback, got, tc.want)
			}
		})
	}
}
