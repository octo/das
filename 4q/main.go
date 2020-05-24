// Package das implements the low-level protocol of Das Keyboard.
//
// This is a very early draft and very much work in progress. The goal is to
// support "Das Keyboard 4Q", because that's the one I happen to own.
//
// For the most part, this is a re-implementation of diefarbe/node-lib.
package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image/color"
	"log"
	"time"

	"github.com/zserge/hid"
)

func main() {
	kb, err := Open()
	if err != nil {
		log.Fatal(err)
	}

	err = kb.send(KeyState{
		LEDID:         6,
		PassiveEffect: SetColor,
		PassiveColor:  color.NRGBA{R: 0x09, G: 0x08, B: 0xF1},
		ActiveEffect:  SetColorActive,
		ActiveColor:   color.NRGBA{R: 0xF2, G: 0x07, B: 0x13},
	})
	if err != nil {
		log.Fatal(err)
	}
	return

	// // INIT? .. looks like it reads a serial number of so.
	// err = kb.setReport([]byte{0xEA, 0x03, 0xB0, 0x59, 0x00, 0x00, 0x00})
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// data, err := kb.dev.GetReport(1)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("<- GetReport() = %#x\n", data)
	// fmt.Println("verifyChecksum():", verifyChecksum(data[:8]))
	// return

	err = kb.setReport([]byte{0xEA, 0x0B, 0x78, 0x03, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x9F, 0x00})
	if err != nil {
		log.Fatal(err)
	}

	data, err := kb.dev.GetReport(1)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("<- GetReport() = %#x\n", data)
	fmt.Println("verifyChecksum():", verifyChecksum(data[:8]))

	err = kb.setReport([]byte{0xEA, 0x08, 0x78, 0x08, 0x05, 0x01, 0x60, 0x61, 0x62, 0xF5, 0x00, 0x00, 0x00, 0x00})
	if err != nil {
		log.Fatal(err)
	}

	err = kb.setReport([]byte{0xEA, 0x0B, 0x78, 0x04, 0x05, 0x1E, 0xFE, 0x01, 0x02, 0x07, 0xD0, 0x00, 0xAC, 0x00})
	//                                                LED   BLINK R     G     B                       CHK
	if err != nil {
		log.Fatal(err)
	}

	data, err = kb.dev.GetReport(1) // this is what fails currently.
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("<- GetReport() = %#x\n", data)

	err = kb.setReport([]byte{0xEA, 0x03, 0x78, 0x0A, 0x9B, 0x00, 0x00})
	if err != nil {
		log.Fatal(err)
	}

	data, err = kb.dev.GetReport(1)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("<- GetReport() = %v\n", data)
}

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
