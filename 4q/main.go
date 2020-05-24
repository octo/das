// Package das implements the low-level protocol of Das Keyboard.
//
// This is a very early draft and very much work in progress. The goal is to
// support "Das Keyboard 4Q", because that's the one I happen to own.
//
// For the most part, this is a re-implementation of diefarbe/node-lib.
package dkb4q

import (
	"errors"
	"fmt"

	"github.com/zserge/hid"
)

const (
	MaxLEDID = 124
)

func (kb *Keyboard) setReport(data []byte) error {
	if len(data) == 14 {
		if err := verifyChecksum(data); err != nil {
			return err
		}

		if err := kb.setReport(data[:7]); err != nil {
			return err
		}
		data = data[7:]
	}
	if len(data) != 7 {
		return fmt.Errorf("len(data) = %d, want 7", len(data))
	}

	fmt.Printf("-> SetReport(1, %x)\n", append([]byte{0x01}, data...))
	return kb.dev.SetReport(1, append([]byte{0x01}, data...))
}

func calculateChecksum(data []byte) (got, want byte) {
	// TODO(octo): the checksum is not necessarily the last non-zero byte.
	// Instead, the message includes a length.
	var lastByte, chkSum byte

	for _, b := range data {
		if b == 0 {
			continue
		}

		chkSum ^= lastByte
		lastByte = b
	}

	return lastByte, chkSum
}

func verifyChecksum(data []byte) error {
	got, want := calculateChecksum(data)
	if got != want {
		return fmt.Errorf("checksum mismatch: got %#x, want %#x", got, want)
	}
	return nil
}

// ErrNotFound is returned by Open if no matching device is found.
var ErrNotFound = errors.New("no Das Keyboard device found")

// Keyboard represents the connection to a keyboard.
type Keyboard struct {
	dev hid.Device
	seq uint8
}

// Open scans USB devices for a "Das Keyboard" by looking for the vendor ID
// 0x24F0. It returns a Keyboard talking to the first device successfully
// opened. If no device could be opened, the last error is returned, or
// ErrNotFound if no matching device was found.
//
// The connection to the keyboard should be closed with Close().
func Open() (Keyboard, error) {
	const vendorID = 0x24F0

	var (
		device  hid.Device
		lastErr error
	)
	hid.UsbWalk(func(dev hid.Device) {
		if device != nil || dev.Info().Vendor != vendorID {
			return
		}

		fmt.Println("dev.Info() =", dev.Info())
		if dev.Info().Interface != 1 {
			return
		}

		if err := dev.Open(); err != nil {
			lastErr = err
			return
		}

		device = dev
		return
	})

	if device == nil {
		if lastErr != nil {
			return Keyboard{}, lastErr
		}
		return Keyboard{}, fmt.Errorf("no DasKeyboard device found")
	}

	kb := Keyboard{
		dev: device,
	}

	return kb, nil
}

// Close closes the connection to the keyboard.
func (kb *Keyboard) Close() error {
	defer func() {
		kb.dev = nil
	}()

	if kb.dev == nil {
		return fmt.Errorf("connection to keyboard not open")
	}

	kb.dev.Close()
	return nil
}
