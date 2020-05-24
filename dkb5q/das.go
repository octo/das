// Package das implements the low-level protocol of Das Keyboard 5Q.
//
// For the most part, this is a re-implementation of diefarbe/node-lib.
package das

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image/color"

	"github.com/zserge/hid"
)

// ErrNotFound is returned by Open if no matching device is found.
var ErrNotFound = errors.New("no Das Keyboard device found")

// Keyboard represents the connection to a keyboard.
type Keyboard struct {
	dev hid.Device
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

	if err := kb.initialize(); err != nil {
		return Keyboard{}, err
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

// initPacket was copied verbatim from diefarbe/node-lib.
//
// At this point we're really not sure exactly what this does.
// However, we know it's very important. This is sent by the main service module.
var initPacket = []byte{
	0x00, 0x13, 0x00, 0x4d, 0x43, 0x49, 0x51, 0x46,
	0x49, 0x46, 0x45, 0x44, 0x4c, 0x48, 0x39, 0x46,
	0x34, 0x41, 0x45, 0x43, 0x58, 0x39, 0x31, 0x36,
	0x50, 0x42, 0x44, 0x35, 0x50, 0x33, 0x41, 0x33,
	0x30, 0x37, 0x38,
}

// initialize initializes the keyboard by sending a magic byte sequence.
func (kb Keyboard) initialize() error {
	const reportID = 0
	return kb.dev.SetReport(reportID, initPacket)
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
		const reportID = 0
		if err := kb.dev.SetReport(reportID, pkg); err != nil {
			return err
		}
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
		uint8(0),
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
