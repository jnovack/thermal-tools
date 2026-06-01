package main

import (
	"math"
	"testing"
)

func TestMmToPx(t *testing.T) {
	t.Parallel()

	// NOTE: 203 DPI thermal printers use approximately 8 dots/mm (exact is
	// 203/25.4 ≈ 7.992 dots/mm). The M220 physical print head has exactly
	// 560 dots across 70mm (8 dots/mm), so phomemo.PrintWidthPx = 560 is the
	// authoritative constant for full-width jobs; mmToPx(70, 203) = 559.
	tests := []struct {
		name   string
		mm     float64
		dpi    int
		wantPx int
	}{
		// At 203 DPI: 1mm = 203/25.4 ≈ 7.992 dots → rounded
		{"70mm@203dpi", 70.0, 203, 559},             // 70*7.992 = 559.45 → 559
		{"40mm@203dpi", 40.0, 203, 320},             // 40*7.992 = 319.69 → 320
		{"30mm@203dpi", 30.0, 203, 240},             // 30*7.992 = 239.76 → 240
		{"25mm@203dpi", 25.0, 203, 200},             // 25*7.992 = 199.80 → 200
		{"10mm@203dpi rounds to 80", 10.0, 203, 80}, // 79.92 → 80
		// Exact: 1 inch = 25.4mm
		{"25.4mm@96dpi is exactly 96px", 25.4, 96, 96},
		{"25.4mm@300dpi is exactly 300px", 25.4, 300, 300},
		// Edge / zero cases
		{"zero mm returns 0", 0.0, 203, 0},
		{"negative mm returns 0", -5.0, 203, 0},
		{"zero dpi returns 0", 70.0, 0, 0},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := mmToPx(tc.mm, tc.dpi)
			if got != tc.wantPx {
				t.Fatalf("mmToPx(%.1f, %d) = %d, want %d", tc.mm, tc.dpi, got, tc.wantPx)
			}
		})
	}
}

func TestMmToPxRoundTrip(t *testing.T) {
	t.Parallel()

	// For any reasonable mm value, converting to px and back should be
	// within ±1 px of the expected value.
	const dpi = 203
	for _, mm := range []float64{10, 20, 30, 40, 50, 60, 70} {
		mm := mm
		t.Run("", func(t *testing.T) {
			t.Parallel()
			px := mmToPx(mm, dpi)
			expectedPx := math.Round(mm * float64(dpi) / 25.4)
			if math.Abs(float64(px)-expectedPx) > 1 {
				t.Fatalf("mmToPx(%.0f, %d) = %d, expected ~%.0f", mm, dpi, px, expectedPx)
			}
		})
	}
}
