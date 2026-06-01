// Package phomemo is a BLE driver for Phomemo M220 (and M110/M120) label
// printers. The M220 is a 70 mm wide label printer at 203 DPI.
//
// Protocol reference: vivier/phomemo-tools cups/filter/rastertopm110.py
// Transport reference: sgrankin/phomemo (M02X BLE implementation)
//
// IMPORTANT hardware notes:
//   - Use Write Without Response only (ATT opcode 0x52). Write Request
//     causes silent failures on M02X; assumed same for M220 family.
//   - Cap chunks at 182 bytes; larger chunks silently corrupt the job.
//   - Add 60 ms between each chunk to avoid buffer overruns.
//   - The M220 advertises as "Mr.inM220" or "M220" depending on firmware.
//   - macOS hides device MAC addresses; always match by LocalName.
package phomemo

// BLE GATT characteristic UUIDs shared across the Phomemo family.
// Verified on M02X (sgrankin/phomemo); expected same for M220 family.
const (
	writeCharUUID  = "0000ff02-0000-1000-8000-00805f9b34fb"
	notifyCharUUID = "0000ff03-0000-1000-8000-00805f9b34fb"

	// DefaultDeviceName is matched against BLE advertised local name.
	// The M220 may advertise as "Mr.inM220" (vivier backend) or "M220".
	DefaultDeviceName = "M220"

	// PrintWidthPx is the M220 print width: 70 mm × 8 dots/mm = 560 pixels.
	PrintWidthPx = 560

	// DPI is the M220 print resolution.
	DPI = 203

	// maxChunkBytes is the maximum BLE write size observed to be safe.
	maxChunkBytes = 182
)

// MediaType controls how the printer detects label boundaries.
type MediaType byte

const (
	// MediaLabelWithGaps uses gap sensing between labels (default).
	MediaLabelWithGaps MediaType = 0x0A
	// MediaContinuous disables label detection for continuous rolls.
	MediaContinuous MediaType = 0x0B
	// MediaLabelWithMarks uses black-mark sensing.
	MediaLabelWithMarks MediaType = 0x26
)

// header returns the ESC/POS print setup sequence for the M220.
// Speed 5 and density 10 are the defaults from the CUPS PPD filter.
func header(mediaType MediaType) []byte {
	return []byte{
		0x1B, 0x4E, 0x0D, 0x05, // select speed = 5
		0x1B, 0x4E, 0x04, 0x0A, // select density = 10
		0x1F, 0x11, byte(mediaType), // select media type
	}
}

// rasterCmd returns the GS v 0 raster command for widthBytes columns
// and heightLines rows of 1bpp raster data.
func rasterCmd(widthBytes, heightLines uint16) []byte {
	return []byte{
		0x1D, 0x76, 0x30, 0x00, // GS v 0, mode=normal
		byte(widthBytes), byte(widthBytes >> 8), // bytes per line, LE
		byte(heightLines), byte(heightLines >> 8), // lines, LE
	}
}

// footer returns the M220 post-print flush commands.
func footer() []byte {
	return []byte{
		0x1F, 0xF0, 0x05, 0x00,
		0x1F, 0xF0, 0x03, 0x00,
	}
}
