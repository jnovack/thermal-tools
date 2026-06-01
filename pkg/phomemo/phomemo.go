package phomemo

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"strings"
	"time"

	"tinygo.org/x/bluetooth"
)

// Verbose enables hex logging of every BLE write and notification to stderr.
var Verbose bool

// Printer is a connected Phomemo M220 printer.
// Obtain one via Open; release with Close.
type Printer struct {
	device   bluetooth.Device
	writeCh  bluetooth.DeviceCharacteristic
	notifyCh bluetooth.DeviceCharacteristic
	mtu      int

	// Width is the raster print width in pixels. Defaults to PrintWidthPx (560).
	Width int

	// MediaType controls label gap/mark detection. Defaults to MediaLabelWithGaps.
	MediaType MediaType

	// complete fires when the printer signals print completion.
	complete chan struct{}
}

// Open scans for a Phomemo M220 by BLE advertised local name and connects.
//
// name is matched exactly, or by "Mr.in" prefix if name is DefaultDeviceName
// ("M220") — the M220 may advertise as "Mr.inM220" depending on firmware.
// Cancel ctx to abort the scan.
func Open(ctx context.Context, name string) (*Printer, error) {
	adapter := bluetooth.DefaultAdapter
	if err := adapter.Enable(); err != nil {
		return nil, fmt.Errorf("enable bluetooth adapter: %w", err)
	}

	addr, err := findDevice(ctx, adapter, name)
	if err != nil {
		return nil, err
	}

	dev, err := adapter.Connect(addr, bluetooth.ConnectionParams{})
	if err != nil {
		return nil, fmt.Errorf("connect to %s: %w", addr.String(), err)
	}

	wch, nch, err := findChars(dev)
	if err != nil {
		_ = dev.Disconnect()
		return nil, err
	}

	mtu, _ := wch.GetMTU()
	if mtu == 0 {
		mtu = 23
	}
	chunk := int(mtu) - 3
	if chunk > maxChunkBytes {
		chunk = maxChunkBytes
	}

	p := &Printer{
		device:    dev,
		writeCh:   wch,
		notifyCh:  nch,
		mtu:       chunk,
		Width:     PrintWidthPx,
		MediaType: MediaLabelWithGaps,
		complete:  make(chan struct{}, 1),
	}

	_ = nch.EnableNotifications(func(buf []byte) {
		if Verbose {
			fmt.Fprintf(os.Stderr, "← notify %d B: % x\n", len(buf), buf)
		}
		// M220 print-complete notification — byte pattern may differ from M02X.
		// Watch for 0x1A 0x0F 0x0C (M02X) or any trailing frame after raster data.
		if len(buf) >= 3 && buf[0] == 0x1A && buf[1] == 0x0F && buf[2] == 0x0C {
			select {
			case p.complete <- struct{}{}:
			default:
			}
		}
	})

	return p, nil
}

// Close disconnects from the printer.
func (p *Printer) Close() error {
	return p.device.Disconnect()
}

// PrintImage converts img to 1bpp and sends it to the printer. The image is
// assumed to already be scaled to the target print width — call
// render.FitWidth before PrintImage. Dithering should be applied
// before calling as well.
//
// PrintImage blocks until the printer notifies print-complete or a timeout
// expires. Disconnecting before completion causes the final rows to drop
// silently on some firmware versions.
func (p *Printer) PrintImage(img image.Image) error {
	bpl := p.Width / 8
	if p.Width%8 != 0 {
		return fmt.Errorf("print width %d not a multiple of 8", p.Width)
	}

	bits, h := pack1bpp(img, p.Width)
	if h > 0xFFFF {
		return fmt.Errorf("image too tall: %d lines (max 65535)", h)
	}

	// Drain any stale completion signal from a previous print.
	select {
	case <-p.complete:
	default:
	}

	hdr := header(p.MediaType)
	raster := rasterCmd(uint16(bpl), uint16(h))
	foot := footer()

	payload := make([]byte, 0, len(hdr)+len(raster)+len(bits)+len(foot))
	payload = append(payload, hdr...)
	payload = append(payload, raster...)
	payload = append(payload, bits...)
	payload = append(payload, foot...)

	if err := p.writeData(payload); err != nil {
		return err
	}

	// Wait for print-complete notification. 45 s covers very tall labels.
	select {
	case <-p.complete:
	case <-time.After(45 * time.Second):
		// Timeout is not fatal; the job may have already completed silently.
	}
	return nil
}

// writeData sends buf via GATT Write Without Response in maxChunkBytes chunks
// with 60 ms inter-chunk pacing to prevent buffer overruns.
func (p *Printer) writeData(buf []byte) error {
	chunk := p.mtu
	if chunk < 20 {
		chunk = 20
	}
	for off := 0; off < len(buf); off += chunk {
		end := off + chunk
		if end > len(buf) {
			end = len(buf)
		}
		if Verbose {
			c := buf[off:end]
			limit := len(c)
			if limit > 32 {
				limit = 32
			}
			suffix := ""
			if len(c) > 32 {
				suffix = "..."
			}
			fmt.Fprintf(os.Stderr, "↦ %d B: % x%s\n", len(c), c[:limit], suffix)
		}
		if _, err := p.writeCh.WriteWithoutResponse(buf[off:end]); err != nil {
			return fmt.Errorf("ble write (%d B at offset %d): %w", end-off, off, err)
		}
		time.Sleep(60 * time.Millisecond)
	}
	return nil
}

func findDevice(ctx context.Context, adapter *bluetooth.Adapter, name string) (bluetooth.Address, error) {
	if name == "" {
		name = DefaultDeviceName
	}

	type result struct {
		addr bluetooth.Address
		err  error
	}
	ch := make(chan result, 1)

	go func() {
		err := adapter.Scan(func(a *bluetooth.Adapter, r bluetooth.ScanResult) {
			adv := r.LocalName()
			matched := adv == name ||
				// Some firmware advertises "Mr.inM220" instead of "M220".
				(name == DefaultDeviceName && strings.HasPrefix(adv, "Mr.in"))
			if matched {
				_ = a.StopScan()
				select {
				case ch <- result{addr: r.Address}:
				default:
				}
			}
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			select {
			case ch <- result{err: err}:
			default:
			}
		}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			return bluetooth.Address{}, r.err
		}
		return r.addr, nil
	case <-ctx.Done():
		_ = adapter.StopScan()
		return bluetooth.Address{}, fmt.Errorf("device %q not found within scan window: %w", name, ctx.Err())
	}
}

func findChars(dev bluetooth.Device) (write, notify bluetooth.DeviceCharacteristic, err error) {
	wantW, _ := bluetooth.ParseUUID(writeCharUUID)
	wantN, _ := bluetooth.ParseUUID(notifyCharUUID)

	srvcs, err := dev.DiscoverServices(nil)
	if err != nil {
		return write, notify, fmt.Errorf("discover services: %w", err)
	}

	var foundW, foundN bool
	for _, s := range srvcs {
		chars, cerr := s.DiscoverCharacteristics(nil)
		if cerr != nil {
			continue
		}
		for _, c := range chars {
			switch c.UUID() {
			case wantW:
				write, foundW = c, true
			case wantN:
				notify, foundN = c, true
			}
		}
	}
	if !foundW {
		return write, notify, errors.New("write characteristic 0xff02 not found — is this a Phomemo device?")
	}
	if !foundN {
		return write, notify, errors.New("notify characteristic 0xff03 not found — is this a Phomemo device?")
	}
	return write, notify, nil
}

// pack1bpp converts img to 1-bit-per-pixel packed bytes, MSB-first per row,
// with 1=black (heat). The image is read at width pixels; pixels beyond
// img.Bounds().Dx() are treated as white.
func pack1bpp(img image.Image, width int) ([]byte, int) {
	b := img.Bounds()
	h := b.Dy()
	bpl := width / 8
	out := make([]byte, bpl*h)

	for y := 0; y < h; y++ {
		for x := 0; x < width && x < b.Dx(); x++ {
			r, g, bv, a := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			if a == 0 {
				continue
			}
			lum := (299*r + 587*g + 114*bv) / 1000
			if lum < 0x8000 { // darker than mid-gray → print as black
				out[y*bpl+x/8] |= 1 << (7 - uint(x%8))
			}
		}
	}
	return out, h
}

// RenderGray converts a paletted or color image to an 8-bit grayscale image,
// used primarily in tests to inspect pack1bpp output.
func RenderGray(img image.Image) *image.Gray {
	b := img.Bounds()
	out := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.Set(x, y, color.GrayModel.Convert(img.At(x, y)))
		}
	}
	return out
}
