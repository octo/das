package dkb4q

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

func (kb *Keyboard) State(s KeyState) error {
	msg0 := encodeReport(0xEA, []byte{0x78, 0x03, s.LEDID, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	if err := kb.setReport(msg0); err != nil {
		return err
	}

	res0, err := getReports(kb.dev)
	if err != nil {
		return err
	}
	// should return "ED 03 78 00 96"
	fmt.Printf("response 0 = %#v\n", res0)

	msg1 := encodeReport(0xEA, []byte{0x78, 0x08, s.LEDID, byte(s.PassiveEffect),
		s.PassiveColor.R, s.PassiveColor.G, s.PassiveColor.B})
	if err := kb.setReport(msg1); err != nil {
		return err
	}

	msg2 := []byte{0x78, 0x04, s.LEDID, byte(s.ActiveEffect),
		s.ActiveColor.R, s.ActiveColor.G, s.ActiveColor.B,
		0x07, 0xD0, 0x00} // TODO(octo): appears to be effect specific
	msg2 = encodeReport(0xEA, msg2)
	if err := kb.setReport(msg2); err != nil {
		return err
	}

	res1, err := getReports(kb.dev)
	if err != nil {
		return err
	}
	// should return "ED 03 78 00 96"
	fmt.Printf("response 1 = %#v\n", res1)

	msg3 := encodeReport(0xEA, []byte{0x78, 0x0A})
	if err := kb.setReport(msg3); err != nil {
		return err
	}

	res2, err := getReports(kb.dev)
	if err != nil {
		return err
	}
	// should return "ED 03 78 00 96"
	fmt.Printf("response 2 = %#v\n", res2)

	return nil
}