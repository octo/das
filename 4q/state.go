package main

import (
	"fmt"
	"image/color"
)

type KeyState struct {
	LEDID         uint8
	PassiveEffect Effect
	PassiveColor  color.NRGBA
	ActiveEffect  Effect
	ActiveColor   color.NRGBA
}

type Effect uint8

const (
	None           Effect = 0x00
	SetColor              = 0x01
	SetColorActive        = 0x1E
	Breathe               = 0x08
	Blink                 = 0x1F
	ColorCycle            = 0x14
)

var packets = [][]byte{
	{0xEA, 0x0B, 0x78, 0x03, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x9F, 0x00},
	{0xEA, 0x08, 0x78, 0x08, 0x05, 0x01, 0x60, 0x61, 0x62, 0xF5, 0x00, 0x00, 0x00, 0x00},
	{0xEA, 0x0B, 0x78, 0x04, 0x05, 0x1E, 0xFE, 0x01, 0x02, 0x07, 0xD0, 0x00, 0xAC, 0x00},
	{0xEA, 0x03, 0x78, 0x0A, 0x9B, 0x00, 0x00},
}

func (kb *Keyboard) send(s KeyState) error {
	buffer := make([]byte, 14, 14)

	// msg 0
	copy(buffer, packets[0])
	buffer[4] = s.LEDID
	buffer[12] = xorAll(buffer[:12])
	if err := kb.setReport(buffer); err != nil {
		return err
	}

	// should return "ED 03 78 00 96"
	ret, err := kb.dev.GetReport(1)
	if err != nil {
		return err
	}
	if err := verifyChecksum(ret); err != nil {
		fmt.Printf("ret=%+v, err=%v\n", ret, err)
	}

	// msg 1
	copy(buffer, packets[1])
	buffer[4] = s.LEDID
	buffer[5] = byte(s.PassiveEffect)
	buffer[6] = s.PassiveColor.R
	buffer[7] = s.PassiveColor.G
	buffer[8] = s.PassiveColor.B
	buffer[9] = xorAll(buffer[:9])
	if err := kb.setReport(buffer); err != nil {
		return err
	}

	// msg 2
	copy(buffer, packets[2])
	buffer[4] = s.LEDID
	buffer[5] = byte(s.ActiveEffect)
	buffer[6] = s.ActiveColor.R
	buffer[7] = s.ActiveColor.G
	buffer[8] = s.ActiveColor.B
	buffer[12] = xorAll(buffer[:12])
	if err := kb.setReport(buffer); err != nil {
		return err
	}

	// should return "ED 03 78 00 96"
	ret, err = kb.dev.GetReport(1)
	if err != nil {
		return err
	}
	if err := verifyChecksum(ret); err != nil {
		fmt.Printf("ret=%+v, err=%v\n", ret, err)
	}

	// msg 2
	if err := kb.setReport(packets[3]); err != nil {
		return err
	}

	// should return "" (zero bytes)
	_, err = kb.dev.GetReport(1)
	return err
}

func xorAll(data []byte) byte {
	var ret byte
	for _, d := range data {
		ret ^= d
	}
	return ret
}
