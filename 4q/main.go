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

	// if err := kb.initialize(); err != nil {
	// 	return Keyboard{}, err
	// }

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

// initPacket was copied verbatim from diefarbe/node-lib.
//
// At this point we're really not sure exactly what this does.
// However, we know it's very important. This is sent by the main service module.
var initPacket = []byte{
	0x00, 0x13, 0x00, 0x4d, 0x43, 0x49, 0x51, 0x46,
	0x49, 0x46, 0x45, 0x44, 0x4c, 0x48, 0x39, 0x46,
	0x34, 0x41, 0x45, 0x43, 0x58, 0x39, 0x31, 0x36,
	0x50, 0x42, 0x44, 0x35, 0x50, 0x33, 0x41, 0x33,
	0x30, 0x37, 0x38, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
}

// initialize initializes the keyboard by sending a magic byte sequence.
func (kb Keyboard) initialize() error {
	return kb.sendPacket(initPacket)
}

// KeyColor sets the color of a single key, identified by key ID, aka. LED ID.
// c's alpha channel (c.A) is ignored.
//
// TODO(octo): read KeyInfo how key IDs / LED IDs are determined.
func (kb Keyboard) KeyColor(c color.NRGBA, ledIDs ...uint8) error {
	s := newKeyState()
	s.ledIDs = ledIDs

	// TODO(octo): this mapping is not true for all keys.
	s.colors[0].toValue = c.R
	s.colors[1].toValue = c.G
	s.colors[2].toValue = c.B

	pkgs, err := s.marshalPackets()
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		if err := kb.sendPacket(pkg); err != nil {
			return err
		}
	}

	return nil
}

func (kb *Keyboard) sendPacket(req []byte) error {
	// pad packets to be 64 bytes.
	if len(req) < 64 {
		padding := bytes.Repeat([]byte{0}, 64-len(req))
		req = append(req, padding...)
	}
	fmt.Printf("len(req) = %d\n", len(req))

	req[2] = byte(kb.seq)
	kb.seq++
	// TODO(octo): pad with zeros to get to len(req) == 64?

	const reportID = 0
	fmt.Printf("SetReport(%d, %v)\n", reportID, req)

	if err := kb.dev.SetReport(reportID, req); err != nil {
		return fmt.Errorf("SetReport(%d, %v) = %w", reportID, req, err)
	}

	const inputPacketSize = -1
	res, err := kb.dev.Read(inputPacketSize, time.Second)
	if err != nil {
		return fmt.Errorf("Read(%d, %v) = %w", inputPacketSize, time.Second, err)
	}
	fmt.Printf("Read() = %v\n", res)

	if res[2] != 0x14 || res[3] != req[2] {
		return fmt.Errorf("did not receive ACK for sequence %d: %v", req[2], res)
	}

	return nil
}

type keyState struct {
	// some keys, such as the space bar, right shift, … map to more than one LED.
	ledIDs []uint8
	colors [3]struct { // 0 = red, 1 = green, 2 = blue
		channelID uint8
		toValue   uint8
		fromValue uint8
	}
	effectFlag effectFlag // default: incrementOnly
	// up
	upIncrement      uint16
	upIncrementDelay uint16
	upHoldLevel      uint16
	upHoldDelay      uint16
	// down
	downDecrementDelay uint16
	downDecrement      uint16
	downHoldLevel      uint16
	downHoldDelay      uint16
	// other
	startDelay uint16
	effectID   uint8 // default: 2
}

func newKeyState() keyState {
	return keyState{
		colors: [3]struct {
			channelID uint8
			toValue   uint8
			fromValue uint8
		}{
			{channelID: 0},
			{channelID: 1},
			{channelID: 2},
		},
		effectFlag: incrementOnly,
		effectID:   2,
	}
}

func (ks keyState) marshalPackets() ([][]byte, error) {
	var packets [][]byte
	for _, ledID := range ks.ledIDs {
		for i := 0; i < 3; i++ {
			pkg, err := ks.marshalPacket(ledID, i)
			if err != nil {
				return nil, err
			}

			packets = append(packets, pkg)
		}
	}

	return packets, nil
}

func (ks keyState) marshalPacket(ledID uint8, colorIndex int) ([]byte, error) {
	const setKeyStateCommand = uint8(0x28)

	var (
		buf bytes.Buffer
		bo  = binary.LittleEndian
	)

	for _, v := range []interface{}{
		uint8(0),
		setKeyStateCommand,
		uint8(0), // seq, will be overwritten by sendPacket
		ks.colors[colorIndex].channelID,
		uint8(1), // wtf is this
		uint8(ledID),
		ks.effectID,
		ks.colors[colorIndex].toValue, ks.upIncrement, ks.upIncrementDelay, ks.upHoldLevel, ks.upHoldDelay,
		ks.colors[colorIndex].fromValue, ks.downDecrement, ks.downDecrementDelay, ks.downHoldLevel, ks.downHoldDelay,
		ks.startDelay,
		uint16(0),
		ks.effectFlag,
	} {
		if err := binary.Write(&buf, bo, v); err != nil {
			return nil, err
		}
	}

	// pad packets to be 64 bytes.
	if l := buf.Len(); l < 64 {
		buf.Write(bytes.Repeat([]byte{0}, 64-l))
	}
	fmt.Printf("buf.Len() = %d", buf.Len())

	return buf.Bytes(), nil
}

type effectFlag uint16

const (
	incrementOnly      effectFlag = 1
	decrementOnly                 = 2
	incrementDecrement            = 25
	decrementIncrement            = 26
	onApplyFlag                   = 16384 // == 1<<14
	transitionFlag                = 4096  // == 1<<12
)

func (f *effectFlag) triggerOnApply() {
	*f = *f | onApplyFlag
}

func (f *effectFlag) triggerNow() {
	// clear onApplyFlag
	nowFlag := effectFlag(1) ^ onApplyFlag
	*f = *f & nowFlag
}

func (f *effectFlag) enableTransition() {
	*f = *f | transitionFlag
}

func (f *effectFlag) disableTransition() {
	// clear transitionFlag
	noTransitionFlag := effectFlag(1) ^ transitionFlag
	*f = *f & noTransitionFlag
}
