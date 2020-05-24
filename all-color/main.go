// all-color sets all keys on the keyboard to the same color.
//
// Currently the colors are hard-coded. Eventually, they should be provided by
// a flag.
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
	"github.com/octo/das/4q"
)

func main() {
	kb, err := 4q.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer kb.Close()

	for i := 0; i < q4.MaxLEDID; i++ {
		err := kb.State(4q.KeyState{
			LEDID:         uint8(i),
			PassiveEffect: SetColor,
			PassiveColor:  color.NRGBA{R: 0x01, G: 0x02, B: 0xF3},
			ActiveEffect:  SetColorActive,
			ActiveColor:   color.NRGBA{R: 0xF4, G: 0x05, B: 0x06},
		})
		if err != nil {
			log.Fatal(err)
		}
	}
}
